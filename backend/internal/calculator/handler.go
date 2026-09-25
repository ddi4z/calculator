package calculator

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
)

const invalidOperationMessage = "Operation must be one of: add, subtract, multiply, divide, power, sqrt, percent"

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
			writeError(w, http.StatusMethodNotAllowed, "INVALID_METHOD", "method not allowed")
			return
		}
		handleCalculate(w, r, calculator)
	})
	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "INVALID_METHOD", "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleCalculate(w http.ResponseWriter, r *http.Request, calculator Calculator) {
	var request calculateRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "request body must be valid JSON")
		return
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "request body must contain one JSON value")
		return
	}

	if !request.operation.present || request.operation.null {
		if request.operation.null {
			writeError(w, http.StatusBadRequest, "INVALID_TYPE", "operation must be a string")
		} else {
			writeError(w, http.StatusBadRequest, "MISSING_FIELD", "operation is required")
		}
		return
	}
	var operation string
	if err := json.Unmarshal(request.operation.raw, &operation); err != nil || operation == "" {
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_TYPE", "operation must be a string")
		} else {
			writeError(w, http.StatusBadRequest, "INVALID_OPERATION", invalidOperationMessage)
		}
		return
	}
	if !supportedOperation(operation) {
		writeError(w, http.StatusBadRequest, "INVALID_OPERATION", invalidOperationMessage)
		return
	}
	if !request.operands.present {
		writeError(w, http.StatusBadRequest, "MISSING_FIELD", "operands is required")
		return
	}
	if request.operands.null {
		writeError(w, http.StatusBadRequest, "INVALID_TYPE", "operands must be an array")
		return
	}
	var rawOperands []json.RawMessage
	if err := json.Unmarshal(request.operands.raw, &rawOperands); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_TYPE", "operands must be an array")
		return
	}
	arity := operationArity(operation)
	if len(rawOperands) > arity {
		writeError(w, http.StatusBadRequest, "INVALID_ARITY", "too many operands")
		return
	}
	if len(rawOperands) < arity {
		writeError(w, http.StatusBadRequest, "MISSING_FIELD", "a required operand is missing")
		return
	}
	operands := make([]float64, arity)
	for i, raw := range rawOperands {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			writeError(w, http.StatusBadRequest, "INVALID_TYPE", "operand must be a number")
			return
		}
		value, err := parseFiniteNumber(raw)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "INVALID_NUMBER", "operand must be a finite number")
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
			message = invalidOperationMessage
		}
		writeError(w, status, ErrorCode(err), message)
		return
	}
	if math.IsNaN(result) || math.IsInf(result, 0) {
		writeError(w, http.StatusUnprocessableEntity, CodeNonFiniteResult, "calculation produced a non-finite result")
		return
	}
	writeJSON(w, http.StatusOK, map[string]float64{"result": result})
}

func parseFiniteNumber(raw json.RawMessage) (float64, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || (trimmed[0] != '-' && (trimmed[0] < '0' || trimmed[0] > '9')) {
		return 0, errors.New("not a JSON number")
	}
	value, err := strconv.ParseFloat(string(trimmed), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, errors.New("not a finite number")
	}
	return value, nil
}

func supportedOperation(name string) bool {
	return name == "add" || name == "subtract" || name == "multiply" || name == "divide" ||
		name == "power" || name == "sqrt" || name == "percent"
}

func operationArity(name string) int {
	if operation, ok := operationRegistry()[name]; ok {
		return operation.Arity()
	}
	return 0
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
				Code:    "INTERNAL_ERROR",
				Message: "unable to encode response",
			},
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}
