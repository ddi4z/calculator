package calculator

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type apiEnvelope struct {
	Result *float64  `json:"result"`
	Error  *apiError `json:"error"`
}

func TestCalculator_InterfaceAndSuccessCases(t *testing.T) {
	calc := NewCalculator()
	if calc == nil {
		t.Fatal("NewCalculator returned nil")
	}

	var _ Calculator = calc

	cases := []struct {
		name      string
		operation string
		operands  []float64
		want      float64
	}{
		{name: "add", operation: "add", operands: []float64{12, 3}, want: 15},
		{name: "subtract", operation: "subtract", operands: []float64{12, 3}, want: 9},
		{name: "multiply", operation: "multiply", operands: []float64{12, 3}, want: 36},
		{name: "divide", operation: "divide", operands: []float64{12, 3}, want: 4},
		{name: "power", operation: "power", operands: []float64{2, 3}, want: 8},
		{name: "sqrt", operation: "sqrt", operands: []float64{81}, want: 9},
		{name: "percent", operation: "percent", operands: []float64{200, 15}, want: 30},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := calc.Calculate(tc.operation, tc.operands)
			if err != nil {
				t.Fatalf("Calculate(%q, %v) returned error: %v", tc.operation, tc.operands, err)
			}
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("Calculate(%q, %v) = %v, want %v", tc.operation, tc.operands, got, tc.want)
			}
		})
	}
}

func TestCalculator_PowerMatchesMathPow(t *testing.T) {
	calc := NewCalculator()
	cases := []struct {
		name     string
		operands []float64
		want     float64
	}{
		{name: "integer exponent", operands: []float64{2, 3}, want: math.Pow(2, 3)},
		{name: "fractional exponent", operands: []float64{4, 0.5}, want: math.Pow(4, 0.5)},
		{name: "negative exponent", operands: []float64{2, -2}, want: math.Pow(2, -2)},
		{name: "fractional reciprocal", operands: []float64{8, 1.0 / 3.0}, want: math.Pow(8, 1.0/3.0)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := calc.Calculate("power", tc.operands)
			if err != nil {
				t.Fatalf("power(%v) returned error: %v", tc.operands, err)
			}
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("power(%v) = %v, want %v", tc.operands, got, tc.want)
			}
		})
	}
}

func TestCalculator_ValidationErrors(t *testing.T) {
	calc := NewCalculator()

	cases := []struct {
		name      string
		operation string
		operands  []float64
		wantCode  string
	}{
		{name: "invalid operation", operation: "mod", operands: []float64{10, 2}, wantCode: CodeInvalidOperation},
		{name: "division by zero", operation: "divide", operands: []float64{10, 0}, wantCode: CodeDivisionByZero},
		{name: "negative sqrt", operation: "sqrt", operands: []float64{-1}, wantCode: CodeNegativeInput},
		{name: "non finite result", operation: "power", operands: []float64{0, -1}, wantCode: CodeNonFiniteResult},
		{name: "invalid arity unary", operation: "sqrt", operands: []float64{9, 1}, wantCode: CodeInvalidArity},
		{name: "invalid arity binary", operation: "add", operands: []float64{1}, wantCode: CodeInvalidArity},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := calc.Calculate(tc.operation, tc.operands)
			if err == nil {
				t.Fatalf("Calculate(%q, %v) expected error but got nil", tc.operation, tc.operands)
			}
			if code := ErrorCode(err); code != tc.wantCode {
				t.Fatalf("Calculate(%q, %v) error code = %q, want %q", tc.operation, tc.operands, code, tc.wantCode)
			}
		})
	}
}

func TestCalculator_RejectsMissingOrNullOperands(t *testing.T) {
	calc := NewCalculator()

	if _, err := calc.Calculate("add", nil); err == nil {
		t.Fatal("Calculate(add, nil) should reject missing operands")
	}
	if _, err := calc.Calculate("add", []float64{}); err == nil {
		t.Fatal("Calculate(add, empty operands) should reject missing operands")
	}
	if _, err := calc.Calculate("sqrt", []float64{}); err == nil {
		t.Fatal("Calculate(sqrt, empty operands) should reject missing operand")
	}

	if _, err := calc.Calculate("add", []float64{math.NaN(), 1}); ErrorCode(err) != CodeInvalidNumber {
		t.Fatalf("Calculate(add, non-finite operand) error code = %q, want %q", ErrorCode(err), CodeInvalidNumber)
	}
}

func TestHTTPHealthEndpoint(t *testing.T) {
	h := NewHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/health status = %d, want %d", rr.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("GET /api/health returned invalid JSON: %v", err)
	}
	if response["status"] != "ok" {
		t.Fatalf("GET /api/health = %#v, want status=ok", response)
	}
}

func TestHTTPRejectsUnsupportedMethodsAndRoutes(t *testing.T) {
	h := NewHandler()

	cases := []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "calculate get",
			method:     http.MethodGet,
			target:     "/api/calculate",
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   codeInvalidMethod,
		},
		{
			name:       "health post",
			method:     http.MethodPost,
			target:     "/api/health",
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   codeInvalidMethod,
		},
		{
			name:       "unknown route",
			method:     http.MethodGet,
			target:     "/api/unknown",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantCode == "" && tc.target == "/api/unknown" {
				req := httptest.NewRequest(tc.method, tc.target, nil)
				rr := httptest.NewRecorder()
				h.ServeHTTP(rr, req)
				if rr.Code != tc.wantStatus {
					t.Fatalf("status = %d, want %d; body=%s", rr.Code, tc.wantStatus, rr.Body.String())
				}
				return
			}

			status, rawBody, response, err := doJSONRequest(t, h, tc.method, tc.target, nil)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if status != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", status, tc.wantStatus, rawBody)
			}
			if tc.wantCode != "" && (response.Error == nil || response.Error.Code != tc.wantCode) {
				t.Fatalf("error = %#v, want code %q; body=%s", response.Error, tc.wantCode, rawBody)
			}
		})
	}
}

func TestRequestLoggingMiddleware(t *testing.T) {
	var logs bytes.Buffer
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
	})

	handler := NewHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /api/health status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(logs.String(), "request method=GET path=/api/health status=200") {
		t.Fatalf("log = %q, want method, path, and status", logs.String())
	}
	if !strings.Contains(logs.String(), "duration=") {
		t.Fatalf("log = %q, want duration", logs.String())
	}
	if strings.Contains(logs.String(), "operands") {
		t.Fatalf("log = %q, must not include request operands", logs.String())
	}
}

func TestHTTPCalculate_SuccessAndValidationContract(t *testing.T) {
	h := NewHandler()

	cases := []struct {
		name       string
		method     string
		payload    any
		wantStatus int
		wantCode   string
		wantResult *float64
		wantError  bool
	}{
		{name: "valid add", method: http.MethodPost, payload: map[string]any{"operation": "add", "operands": []float64{12, 3}}, wantStatus: http.StatusOK, wantResult: floatPtr(15)},
		{name: "valid sqrt", method: http.MethodPost, payload: map[string]any{"operation": "sqrt", "operands": []float64{81}}, wantStatus: http.StatusOK, wantResult: floatPtr(9)},
		{name: "invalid operation", method: http.MethodPost, payload: map[string]any{"operation": "mod", "operands": []float64{1, 2}}, wantStatus: http.StatusBadRequest, wantCode: "INVALID_OPERATION", wantError: true},
		{name: "missing operation", method: http.MethodPost, payload: map[string]any{"operands": []float64{1, 2}}, wantStatus: http.StatusBadRequest, wantCode: "MISSING_FIELD", wantError: true},
		{name: "missing operands", method: http.MethodPost, payload: map[string]any{"operation": "add"}, wantStatus: http.StatusBadRequest, wantCode: "MISSING_FIELD", wantError: true},
		{name: "null operation", method: http.MethodPost, payload: map[string]any{"operation": nil, "operands": []float64{1, 2}}, wantStatus: http.StatusBadRequest, wantCode: "INVALID_TYPE", wantError: true},
		{name: "null operands", method: http.MethodPost, payload: map[string]any{"operation": "add", "operands": nil}, wantStatus: http.StatusBadRequest, wantCode: "INVALID_TYPE", wantError: true},
		{name: "null operand element", method: http.MethodPost, payload: map[string]any{"operation": "add", "operands": []any{12, nil}}, wantStatus: http.StatusBadRequest, wantCode: "INVALID_TYPE", wantError: true},
		{name: "missing operand add", method: http.MethodPost, payload: map[string]any{"operation": "add", "operands": []float64{12}}, wantStatus: http.StatusBadRequest, wantCode: "MISSING_FIELD", wantError: true},
		{name: "arity mismatch sqrt", method: http.MethodPost, payload: map[string]any{"operation": "sqrt", "operands": []float64{81, 1}}, wantStatus: http.StatusBadRequest, wantCode: "INVALID_ARITY", wantError: true},
		{name: "division by zero", method: http.MethodPost, payload: map[string]any{"operation": "divide", "operands": []float64{10, 0}}, wantStatus: http.StatusUnprocessableEntity, wantCode: "DIVISION_BY_ZERO", wantError: true},
		{name: "negative sqrt", method: http.MethodPost, payload: map[string]any{"operation": "sqrt", "operands": []float64{-1}}, wantStatus: http.StatusUnprocessableEntity, wantCode: "NEGATIVE_INPUT", wantError: true},
		{name: "non finite result", method: http.MethodPost, payload: map[string]any{"operation": "power", "operands": []float64{0, -1}}, wantStatus: http.StatusUnprocessableEntity, wantCode: "NON_FINITE_RESULT", wantError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, rawBody, body, err := doJSONRequest(t, h, tc.method, "/api/calculate", tc.payload)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if status != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", status, tc.wantStatus, rawBody)
			}

			if tc.wantError {
				if body.Error == nil {
					t.Fatalf("expected error payload, got %s", rawBody)
				}
				if tc.wantCode != "" && !strings.Contains(strings.ToUpper(body.Error.Code), tc.wantCode) {
					t.Fatalf("error code = %q, want code containing %q; body=%s", body.Error.Code, tc.wantCode, rawBody)
				}
				return
			}

			if body.Error != nil {
				t.Fatalf("unexpected error payload: %#v; body=%s", *body.Error, rawBody)
			}
			if body.Result == nil {
				t.Fatal("expected result payload but got nil")
			}
			if tc.wantResult != nil && math.Abs(*body.Result-*tc.wantResult) > 1e-9 {
				t.Fatalf("result = %v, want %v", *body.Result, *tc.wantResult)
			}
		})
	}
}

func TestHTTPCalculate_InvalidJSONAndMissingFields(t *testing.T) {
	h := NewHandler()

	cases := []struct {
		name     string
		payload  string
		wantCode string
	}{
		{name: "malformed object", payload: "{bad json", wantCode: codeInvalidJSON},
		{name: "multiple JSON values", payload: `{"operation":"add","operands":[1,2]} true`, wantCode: codeInvalidJSON},
		{name: "non-object array", payload: `[{"operation":"add","operands":[1,2]}]`, wantCode: codeInvalidType},
		{name: "scalar body", payload: `42`, wantCode: codeInvalidType},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewBufferString(tc.payload))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
			}

			var payload apiEnvelope
			if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
				t.Fatalf("response not parseable: %v", err)
			}
			if payload.Error == nil || payload.Error.Code != tc.wantCode {
				t.Fatalf("error = %#v, want code %q", payload.Error, tc.wantCode)
			}
		})
	}

	t.Run("duplicate and unknown fields", func(t *testing.T) {
		payload := `{"operation":"add","operation":"multiply","operands":[3,4],"unknown":true}`
		status, rawBody, response, err := doRawJSONRequest(t, h, http.MethodPost, "/api/calculate", payload)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if status != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", status, http.StatusOK, rawBody)
		}
		if response.Result == nil || *response.Result != 12 {
			t.Fatalf("result = %#v, want 12; body=%s", response.Result, rawBody)
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		body := map[string]any{"operation": "add"}
		status, rawBody, response, err := doJSONRequest(t, h, http.MethodPost, "/api/calculate", body)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if status != http.StatusBadRequest || response.Error == nil || response.Error.Code != codeMissingField {
			t.Fatalf("status=%d error=%#v, want 400/%s; body=%s", status, response.Error, codeMissingField, rawBody)
		}
	})

	t.Run("null operands", func(t *testing.T) {
		body := `{"operation":"add","operands":null}`
		status, rawBody, response, err := doRawJSONRequest(t, h, http.MethodPost, "/api/calculate", body)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if status != http.StatusBadRequest || response.Error == nil || response.Error.Code != codeInvalidType {
			t.Fatalf("status=%d error=%#v, want 400/%s; body=%s", status, response.Error, codeInvalidType, rawBody)
		}
	})

	t.Run("null operand element", func(t *testing.T) {
		body := `{"operation":"add","operands":[12,null]}`
		status, rawBody, response, err := doRawJSONRequest(t, h, http.MethodPost, "/api/calculate", body)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if status != http.StatusBadRequest || response.Error == nil || response.Error.Code != codeInvalidType {
			t.Fatalf("status=%d error=%#v, want 400/%s; body=%s", status, response.Error, codeInvalidType, rawBody)
		}
	})

	for _, tc := range []struct {
		name    string
		payload string
	}{
		{name: "string operand", payload: `{"operation":"add","operands":[12,"3"]}`},
		{name: "boolean operand", payload: `{"operation":"add","operands":[12,true]}`},
		{name: "object operand", payload: `{"operation":"add","operands":[12,{}]}`},
		{name: "non-finite operand", payload: `{"operation":"add","operands":[12,1e999]}`},
	} {
		t.Run("invalid "+tc.name, func(t *testing.T) {
			status, rawBody, response, err := doRawJSONRequest(t, h, http.MethodPost, "/api/calculate", tc.payload)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if status != http.StatusUnprocessableEntity || response.Error == nil || response.Error.Code != CodeInvalidNumber {
				t.Fatalf("status=%d error=%#v, want 422/%s; body=%s", status, response.Error, CodeInvalidNumber, rawBody)
			}
		})
	}
}

func doJSONRequest(t *testing.T, h http.Handler, method, target string, payload any) (int, string, apiEnvelope, error) {
	t.Helper()

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return 0, "", apiEnvelope{}, err
		}
		body = bytes.NewReader(data)
	}

	return doRequest(t, h, method, target, body)
}

func doRawJSONRequest(t *testing.T, h http.Handler, method, target, payload string) (int, string, apiEnvelope, error) {
	t.Helper()
	return doRequest(t, h, method, target, bytes.NewBufferString(payload))
}

func doRequest(t *testing.T, h http.Handler, method, target string, body io.Reader) (int, string, apiEnvelope, error) {
	t.Helper()

	req := httptest.NewRequest(method, target, body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	rawBody := rr.Body.String()

	var resp apiEnvelope
	if rr.Body.Len() > 0 {
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			return rr.Code, rawBody, apiEnvelope{}, err
		}
	}

	return rr.Code, rawBody, resp, nil
}

func floatPtr(v float64) *float64 {
	return &v
}
