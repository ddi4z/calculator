package calculator

import (
	"bytes"
	"encoding/json"
	"io"
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
	}{
		{name: "invalid operation", operation: "mod", operands: []float64{10, 2}},
		{name: "division by zero", operation: "divide", operands: []float64{10, 0}},
		{name: "negative sqrt", operation: "sqrt", operands: []float64{-1}},
		{name: "non finite result", operation: "power", operands: []float64{0, -1}},
		{name: "invalid arity unary", operation: "sqrt", operands: []float64{9, 1}},
		{name: "invalid arity binary", operation: "add", operands: []float64{1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := calc.Calculate(tc.operation, tc.operands)
			if err == nil {
				t.Fatalf("Calculate(%q, %v) expected error but got nil", tc.operation, tc.operands)
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

	req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	var payload apiEnvelope
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON response not parseable: %v", err)
	}
	if payload.Error == nil || !strings.Contains(strings.ToUpper(payload.Error.Code), "INVALID_JSON") {
		t.Fatalf("invalid JSON response = %#v, want INVALID_JSON error", payload)
	}

	body := map[string]any{"operation": "add", "operands": []any{12, nil}}
	status, rawBody, resp, err := doJSONRequest(t, h, http.MethodPost, "/api/calculate", body)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if status != http.StatusBadRequest {
		t.Fatalf("null element status = %d, want %d; body=%s", status, http.StatusBadRequest, rawBody)
	}
	if resp.Error == nil || !strings.Contains(strings.ToUpper(resp.Error.Code), "INVALID_TYPE") {
		t.Fatalf("null element error = %#v, want INVALID_TYPE; body=%s", resp.Error, rawBody)
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
