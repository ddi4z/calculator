# Calculator App Specification

## 1. Purpose

This document defines the minimum viable implementation for a full-stack calculator application that can be completed within a realistic 2–4 hour window. It is the single source of truth for implementation and is intentionally scoped to a small, reliable system with clear contracts and measurable acceptance criteria.

The solution consists of:
- a React + TypeScript frontend
- a Go backend REST API
- a small set of automated tests covering the most important logic

## 2. Recommended Minimum Scope

### 2.1 In Scope

The project must support the following required operations:
- addition
- subtraction
- multiplication
- division

Optional advanced operations may be included only if they do not add meaningful risk or complexity:
- exponentiation
- square root
- percentage

The recommended minimum implementation is:
- a single calculator UI with two numeric inputs and an operation selector
- a submit button to send the request to the backend
- clear display of the result or validation error
- backend API with one calculation endpoint
- input validation for all numeric values and edge cases
- tests for backend validation and arithmetic logic
- a lightweight frontend smoke test for rendering and interaction

### 2.2 Out of Scope

The following are explicitly not required for the minimum viable build:
- persistent user sessions
- authentication
- database storage
- multi-step calculation history
- advanced scientific calculator UI
- complex animations or design systems
- Docker deployment unless time permits

## 3. System Architecture

### 3.1 Frontend

- Framework: React with TypeScript
- Tooling: Vite
- State handling: local component state using `useState`
- UX model: single-screen calculator
- Data flow: user input -> frontend validation -> backend request -> result display

### 3.2 Backend

- Language: Go
- Framework: standard library `net/http` or a lightweight router (preferred minimal dependency: `net/http` only unless a tiny router is already available)
- API style: REST JSON
- Contract: one primary calculation endpoint

## 4. API Contract

### 4.1 Base URL

- Local backend API base URL: `/api`
- Example: `http://localhost:8080/api`

### 4.2 Health Check Endpoint

#### GET /api/health

Purpose: verifies that the backend is running.

Request:
- no body

Response status: `200 OK`

Response body:
```json
{
  "status": "ok"
}
```

## 5. Calculation Endpoint

### 5.1 POST /api/calculate

Purpose: performs one arithmetic operation using the provided operands.

#### Request format

```json
{
  "operation": "add",
  "a": 12,
  "b": 3
}
```

#### Supported operations

- `add`
- `subtract`
- `multiply`
- `divide`
- `power` (optional)
- `sqrt` (optional)
- `percent` (optional)

#### Validation rules for request payload

- `operation` is required and must be a non-empty string
- `operation` must be in the supported set
- `a` is required for all operations
- `b` is required for binary operations: `add`, `subtract`, `multiply`, `divide`
- `b` is not required for unary operations such as `sqrt` and `percent` if they are supported
- numeric values must be finite numbers
- `NaN`, `Infinity`, and `-Infinity` are rejected
- `divide` rejects `b === 0`
- `sqrt` rejects negative values
- `percent` interprets as `a * b / 100` when implemented
- `power` computes `a ^ b` using exponentiation logic

#### Success response format

```json
{
  "ok": true,
  "operation": "add",
  "a": 12,
  "b": 3,
  "result": 15
}
```

For unary operations, if implemented:

```json
{
  "ok": true,
  "operation": "sqrt",
  "a": 81,
  "result": 9
}
```

#### Error response format

```json
{
  "ok": false,
  "error": {
    "code": "INVALID_OPERATION",
    "message": "Operation must be one of: add, subtract, multiply, divide"
  }
}
```

## 6. HTTP Status Codes and Error Handling

### 6.1 Success
- `200 OK` for valid calculation requests
- `200 OK` for health checks

### 6.2 Client errors
- `400 Bad Request` for malformed JSON, missing fields, invalid types, or unsupported values
- `422 Unprocessable Entity` for semantically invalid data, such as division by zero or invalid numeric input

### 6.3 Server errors
- `500 Internal Server Error` for unexpected server-side failures

### 6.4 Recommended error codes

| Code | Meaning | Typical HTTP status |
|---|---|---|
| `INVALID_JSON` | Request body is not valid JSON | 400 |
| `MISSING_FIELD` | Required field is missing | 400 |
| `INVALID_TYPE` | Field value is wrong type | 400 |
| `INVALID_OPERATION` | Unsupported operation | 400 |
| `DIVISION_BY_ZERO` | Division attempted with zero divisor | 422 |
| `INVALID_NUMBER` | Value is NaN/Infinity or otherwise invalid | 422 |
| `NEGATIVE_INPUT` | Input invalid for sqrt or similar unary op | 422 |
| `INTERNAL_ERROR` | Unexpected backend failure | 500 |

## 7. Validation Rules

### 7.1 Frontend validation

The frontend must validate before sending a request:
- required numeric fields cannot be empty
- values must parse as valid numbers
- `b` must not be zero when division is selected
- invalid states must disable the submit button or show inline errors
- backend errors must be surfaced clearly to the user

### 7.2 Backend validation

The backend must reject requests with:
- missing JSON body
- missing `operation`
- missing operands for binary operations
- invalid operation names
- non-numeric values
- NaN or infinite values
- division by zero
- invalid sqrt input

### 7.3 Number handling

- All numeric values should be parsed as floating-point numbers
- Results may be returned as integers when exact, otherwise as decimal values
- Rounding should be limited to a reasonable precision to avoid meaningless long floats
- Example: `1 / 3` may return `0.3333333333` or a trimmed decimal representation, depending on implementation

## 8. Frontend Requirements

### 8.1 UI

The frontend must provide:
- a numeric input for operand A
- a numeric input for operand B
- an operation selector
- a calculate button
- a display area for the result
- an error message area for invalid input or API failure

### 8.2 Responsive behavior

- layout should work on desktop and small mobile screens
- forms should stack cleanly on narrow viewports
- buttons and inputs must remain usable on touch devices

### 8.3 User experience requirements

- immediate feedback when input is invalid
- disabled submit button when invalid
- clear result display after successful calculation
- consistent error messages matching API validation messages

## 9. Testing Strategy

### 9.1 Backend tests

Use Go’s built-in `testing` package.

Required test coverage:
- addition success
- subtraction success
- multiplication success
- division success
- division by zero rejection
- invalid operation rejection
- invalid payload rejection
- negative sqrt rejection if supported
- health endpoint response

Recommended structure:
- unit tests for each operation
- handler-level tests for HTTP responses and status codes
- validation tests for payload rules

### 9.2 Frontend tests

Use Vitest + React Testing Library.

Required checks:
- renders the calculator form
- allows user input and submit
- displays a valid result for a basic calculation
- shows validation error for invalid input
- displays backend error message appropriately

### 9.3 Manual acceptance testing

- run backend locally
- run frontend locally
- verify basic arithmetic works end-to-end
- verify invalid inputs are blocked
- verify division by zero returns a clean error

## 10. Acceptance Criteria

The project is considered complete when all of the following are true:

1. The frontend loads and shows a calculator interface.
2. A user can compute addition, subtraction, multiplication, and division.
3. The frontend sends requests to the backend and displays results.
4. Invalid inputs are blocked and user-friendly messages are shown.
5. Division by zero is rejected by the backend with a structured error response.
6. The backend exposes a REST JSON API with predictable request/response contracts.
7. Unit tests cover the critical operations and validation paths.
8. Setup instructions are documented in the project README.
9. The codebase is organized cleanly enough for another developer to continue without confusion.

## 11. Design Decisions and Assumptions

### 11.1 Why a single calculation endpoint?

A single endpoint reduces implementation time and keeps the API easy to reason about. It also makes testing simpler and reduces unnecessary complexity for a small assignment.

### 11.2 Why minimal dependencies?

The project should stay lean and fast to implement. Using a minimal Go stack and a simple React UI avoids dependency overhead and keeps the solution easy to maintain.

### 11.3 Why strict validation?

The assignment explicitly calls out invalid inputs and edge cases. Strong validation ensures the API is predictable and makes the frontend/backend contract easier to test.

### 11.4 Assumption: advanced operations are optional

Advanced arithmetic is not required. If included, it should be added only after the required operations are working cleanly.

### 11.5 Assumption: no persistence layer

The assignment does not require storing calculation history or user data, and that is intentionally excluded from the minimum viable scope.

## 12. Implementation Notes for Future Agents

- Build the backend first, then wire the frontend to the API.
- Keep the API contract stable while implementing the UI.
- Prefer correctness and readable code over extra features.
- Do not expand scope beyond the minimum viable requirement unless time remains.
- If advanced operations are added, they should be implemented as additional operation strings in the same request contract rather than as separate endpoint complexity.

## 13. Final Scope Recommendation

The recommended implementation is:
- React frontend with a simple calculator UI
- Go backend with a single `/api/calculate` endpoint
- required operations: add, subtract, multiply, divide
- optional advanced operations: power, sqrt, percent
- validation for all edge cases and malformed inputs
- tests for backend logic and frontend smoke behavior
- documentation for setup and usage

This is the highest-confidence path to a clean, completed project within the designated time budget.
