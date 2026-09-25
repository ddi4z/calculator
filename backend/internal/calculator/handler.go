package calculator

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
)

const (
	codeInvalidJSON   = "INVALID_JSON"
	codeMissingField  = "MISSING_FIELD"
	codeInvalidType   = "INVALID_TYPE"
	codeInvalidMethod = "INVALID_METHOD"
	codeInternalError = "INTERNAL_ERROR"

	messageInvalidJSON       = "request body must be valid JSON"
	messageRequestObject     = "request body must be a JSON object"
	messageMultipleJSON      = "request body must contain one JSON value"
	messageOperationType     = "operation must be a string"
	messageOperationRequired = "operation is required"
	messageOperandsRequired  = "operands is required"
	messageOperandsType      = "operands must be an array"
	messageTooManyOperands   = "too many operands"
	messageMissingOperand    = "a required operand is missing"
	messageOperandType       = "operand must be a number"
	messageInvalidMethod     = "method not allowed"
	messageResponseEncoding  = "unable to encode response"
	messageInvalidJSONNumber = "not a JSON number"
	messageNonFiniteNumber   = "not a finite number"
)

var errRequestBodyType = errors.New("request body must be a JSON object")
var errJSONNumberType = errors.New(messageInvalidJSONNumber)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error APIError `json:"error"`
}

type presentValue struct {
	present bool
	null    bool
	raw     json.RawMessage
}

type calculateRequest struct {
	operation presentValue
	operands  presentValue
}

func (r *calculateRequest) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return errRequestBodyType
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	r.operation = valueFromFields(fields, "operation")
	r.operands = valueFromFields(fields, "operands")
	return nil
}

func valueFromFields(fields map[string]json.RawMessage, name string) presentValue {
	raw, ok := fields[name]
	if !ok {
		return presentValue{}
	}
	return presentValue{present: true, null: bytes.Equal(bytes.TrimSpace(raw), []byte("null")), raw: raw}
}

func NewHandler(calculators ...Calculator) http.Handler {
	var calculator Calculator = NewCalculator()
	if len(calculators) > 0 && calculators[0] != nil {
		calculator = calculators[0]
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", healthHandler)
	mux.HandleFunc("/api/calculate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, codeInvalidMethod, messageInvalidMethod)
			return
		}
		handleCalculate(w, r, calculator)
	})
	return requestLoggingMiddleware(mux)
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *loggingResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &loggingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		log.Printf("request method=%s path=%s status=%d duration=%s", r.Method, r.URL.Path, status, time.Since(start))
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, codeInvalidMethod, messageInvalidMethod)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleCalculate(w http.ResponseWriter, r *http.Request, calculator Calculator) {
	var request calculateRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		code := codeInvalidJSON
		message := messageInvalidJSON
		if errors.Is(err, errRequestBodyType) {
			code = codeInvalidType
			message = messageRequestObject
		}
		writeError(w, http.StatusBadRequest, code, message)
		return
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, codeInvalidJSON, messageMultipleJSON)
		return
	}

	if !request.operation.present || request.operation.null {
		if request.operation.null {
			writeError(w, http.StatusBadRequest, codeInvalidType, messageOperationType)
		} else {
			writeError(w, http.StatusBadRequest, codeMissingField, messageOperationRequired)
		}
		return
	}
	var operation string
	if err := json.Unmarshal(request.operation.raw, &operation); err != nil || operation == "" {
		if err != nil {
			writeError(w, http.StatusBadRequest, codeInvalidType, messageOperationType)
		} else {
			writeError(w, http.StatusBadRequest, CodeInvalidOperation, MessageInvalidOperation)
		}
		return
	}
	operationDefinition, ok := operationRegistry()[operation]
	if !ok {
		writeError(w, http.StatusBadRequest, CodeInvalidOperation, MessageInvalidOperation)
		return
	}
	if !request.operands.present {
		writeError(w, http.StatusBadRequest, codeMissingField, messageOperandsRequired)
		return
	}
	if request.operands.null {
		writeError(w, http.StatusBadRequest, codeInvalidType, messageOperandsType)
		return
	}
	var rawOperands []json.RawMessage
	if err := json.Unmarshal(request.operands.raw, &rawOperands); err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidType, messageOperandsType)
		return
	}
	arity := operationDefinition.Arity()
	if len(rawOperands) > arity {
		writeError(w, http.StatusBadRequest, CodeInvalidArity, messageTooManyOperands)
		return
	}
	if len(rawOperands) < arity {
		writeError(w, http.StatusBadRequest, codeMissingField, messageMissingOperand)
		return
	}
	operands := make([]float64, arity)
	for i, raw := range rawOperands {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			writeError(w, http.StatusBadRequest, codeInvalidType, messageOperandType)
			return
		}
		value, err := parseFiniteNumber(raw)
		if err != nil {
			if errors.Is(err, errJSONNumberType) {
				writeError(w, http.StatusBadRequest, codeInvalidType, messageOperandType)
				return
			}
			writeError(w, http.StatusUnprocessableEntity, CodeInvalidNumber, messageNonFiniteNumber)
			return
		}
		operands[i] = value
	}
	result, err := calculator.Calculate(operation, operands)
	if err != nil {
		status := http.StatusUnprocessableEntity
		if ErrorCode(err) == CodeInvalidOperation || ErrorCode(err) == CodeInvalidArity {
			status = http.StatusBadRequest
		}
		message := err.Error()
		if ErrorCode(err) == CodeInvalidOperation {
			message = MessageInvalidOperation
		}
		writeError(w, status, ErrorCode(err), message)
		return
	}
	if math.IsNaN(result) || math.IsInf(result, 0) {
		writeError(w, http.StatusUnprocessableEntity, CodeNonFiniteResult, MessageNonFiniteResult)
		return
	}
	writeJSON(w, http.StatusOK, map[string]float64{"result": result})
}

func parseFiniteNumber(raw json.RawMessage) (float64, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || (trimmed[0] != '-' && (trimmed[0] < '0' || trimmed[0] > '9')) {
		return 0, errJSONNumberType
	}
	value, err := strconv.ParseFloat(string(trimmed), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, errors.New(messageNonFiniteNumber)
	}
	return value, nil
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: APIError{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	body, err := json.Marshal(value)
	if err != nil {
		status = http.StatusInternalServerError
		body, _ = json.Marshal(errorResponse{
			Error: APIError{
				Code:    codeInternalError,
				Message: messageResponseEncoding,
			},
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}
