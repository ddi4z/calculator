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
- exponentiation
- square root
- percentage

The recommended minimum implementation is:
- a single calculator UI with operand inputs and an operation selector; the UI
  may render two inputs for binary operations and only one input for unary
  operations
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
  "operands": [12, 3]
}
```

#### Supported operations

- `add`
- `subtract`
- `multiply`
- `divide`
- `power`
- `sqrt`
- `percent`

#### Validation rules for request payload

- `operation` is required and must be a non-empty string
- `operation` must be in the supported set
- `operands` is required and must be a JSON array
- The number of items in `operands` must exactly match the selected operation's
  declared arity
- Unary operations require one operand
- Binary operations require two operands
- JSON fields that are omitted are treated as absent; JSON has no `undefined`
  value
- An omitted required field returns `MISSING_FIELD`
- An explicitly provided `null` value for `operation` or `operands` returns
  `INVALID_TYPE`
- An explicitly provided `null` element inside `operands` returns
  `INVALID_TYPE`
- A JSON array cannot contain an `undefined` or omitted slot. If the array is
  shorter than the selected operation's arity, the missing required operand
  position returns `MISSING_FIELD`
- Every item in `operands` must be a finite number
- `NaN`, `Infinity`, and `-Infinity` are rejected
- `divide` rejects an operands array whose second item is `0`
- `sqrt` rejects an operands array whose first item is negative
- `percent` interprets `operands[0] * operands[1] / 100`
- `power` computes `math.Pow(operands[0], operands[1])` using Go's
  `math.Pow` semantics

#### Success response format

```json
{
  "result": 15
}
```

For the unary `sqrt` operation:

```json
{
  "result": 9
}
```

#### Error response format

```json
{
  "error": {
    "code": "INVALID_OPERATION",
    "message": "Operation must be one of: add, subtract, multiply, divide, power, sqrt, percent"
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
| `MISSING_FIELD` | Required field or operand element is omitted | 400 |
| `INVALID_TYPE` | Field or operand element has an invalid type, including explicit `null` | 400 |
| `INVALID_OPERATION` | Unsupported operation | 400 |
| `INVALID_ARITY` | Operand array contains more items than the operation accepts | 400 |
| `DIVISION_BY_ZERO` | Division attempted with zero divisor | 422 |
| `INVALID_NUMBER` | Value is NaN/Infinity or otherwise invalid | 422 |
| `NON_FINITE_RESULT` | Calculation produced NaN, +Inf, or -Inf | 422 |
| `NEGATIVE_INPUT` | Input invalid for sqrt or similar unary op | 422 |
| `INTERNAL_ERROR` | Unexpected backend failure | 500 |

## 7. Validation Rules

### 7.1 Frontend validation

The frontend must validate before sending a request:
- required numeric fields for the selected operation cannot be empty
- values for visible inputs must parse as valid numbers
- the number of collected operands must match the selected operation's arity
- the second operand must not be zero when division is selected
- when a unary operation is selected, the second operand input must be hidden or
  disabled, its value must be cleared or ignored, and the request must contain
  exactly one item in `operands`
- when a binary operation is selected, both operand inputs must be visible or
  enabled and the request must contain exactly two items in `operands`
- invalid states must disable the submit button or show inline errors
- backend errors must be surfaced clearly to the user

### 7.2 Backend validation

The backend must reject requests with:
- missing JSON body
- missing `operation`
- missing `operands`
- invalid operation names
- an `operands` value that is not an array
- an operands array with extra items, which returns an arity validation error
- an omitted required field, which must return `MISSING_FIELD`
- an explicitly provided `null` request field or operand item, which must
  return `INVALID_TYPE`
- non-numeric values
- NaN or infinite values
- division by zero
- invalid sqrt input

Because the backend is required to use Go, operand-count validation must be
centralized in a typed operation abstraction rather than implemented as
separate conditional checks for every operation. Define a Go interface and
registry equivalent to:

```go
type Operation interface {
    Name() string
    Arity() int
    Evaluate(operands []float64) (float64, error)
}
```

Each operation must implement `Operation` and declare its expected operand
count:

- `Arity() == 1`: requires an `operands` array containing exactly one number
- `Arity() == 2`: requires an `operands` array containing exactly two numbers

Use a presence-aware Go JSON representation, such as a custom type implementing
`json.Unmarshaler` or an equivalent mechanism, so the handler can distinguish
an omitted field from an explicitly provided `null` value. Pointer fields alone
are not sufficient because both an omitted pointer and a JSON `null` value
decode to `nil`.

The representation must track at least:

- whether the field was present
- whether the field was explicitly `null`
- the decoded non-null value

The required `operation` and `operands` fields, and every required operand
position, must be present and non-null. An omitted required field or a
shortened operands array returns `MISSING_FIELD`; an explicit `null` value
returns `INVALID_TYPE`. Extra operand items return an arity validation error.

Store the supported operations in a registry keyed by operation name. Request
handling must:

1. Look up the operation definition.
2. Validate that `operands` is present and non-null, returning `MISSING_FIELD`
   when omitted and `INVALID_TYPE` when explicitly null.
3. Validate that the operands array contains no null elements. A shortened
   array returns `MISSING_FIELD` for the missing required operand position; an
   explicit null element returns `INVALID_TYPE`; extra items return an arity
   validation error.
4. Validate every item in the `operands` array as a finite number.
5. Invoke `Evaluate(operands)`.
6. Validate that the returned result is finite. If the result is `NaN`, `+Inf`,
   or `-Inf`, return a `422 Unprocessable Entity` response with error code
   `NON_FINITE_RESULT`; never serialize the result in a JSON response.

The minimum operation registry is:

| Operation | Arity | Required operands | Formula |
|---|---:|---|---|
| `add` | 2 | `operands[0]`, `operands[1]` | `operands[0] + operands[1]` |
| `subtract` | 2 | `operands[0]`, `operands[1]` | `operands[0] - operands[1]` |
| `multiply` | 2 | `operands[0]`, `operands[1]` | `operands[0] * operands[1]` |
| `divide` | 2 | `operands[0]`, `operands[1]` | `operands[0] / operands[1]` |
| `power` | 2 | `operands[0]`, `operands[1]` | `math.Pow(operands[0], operands[1])` |
| `percent` | 2 | `operands[0]`, `operands[1]` | `operands[0] * operands[1] / 100` |
| `sqrt` | 1 | `operands[0]` | `sqrt(operands[0])` |

Arity must be declared once in the operation implementation and enforced by
shared request-validation code. Adding a new operation should require adding
one new `Operation` implementation and registering it, without modifying the
central handler's operand-count or evaluation-call logic. `Evaluate` must
validate the slice length defensively against the operation's expected arity
before accessing any element.

### 7.3 Number handling

- All numeric values should be parsed as floating-point numbers
- The backend must compute and return results using standard `float64` JSON encoding
- The backend must never return `NaN`, `+Inf`, or `-Inf` in a JSON response
- Every calculation result must pass a finiteness check before serialization;
  non-finite results return `NON_FINITE_RESULT` with HTTP `422`
- The backend must not apply application-specific rounding or truncate decimal results
- The frontend is responsible for presentation formatting, including limiting displayed decimal places
- Example: `1 / 3` is returned using the backend's standard `float64` JSON representation, while the frontend may display a rounded value such as `0.3333`

## 8. Frontend Requirements

### 8.1 UI

The frontend must provide:
- a numeric input for the first item in `operands`
- a numeric input for the second item in `operands` when a binary operation is
  selected
- an operation selector
- a calculate button
- a display area for the result
- an error message area for invalid input or API failure

The operation selector controls the operand form:

- Binary operations (`add`, `subtract`, `multiply`, `divide`, `power`, and
  `percent`) display or enable both operand inputs and submit
  `operands: [first, second]`.
- Unary operations (`sqrt`) display or enable only the first operand input and
  submit `operands: [first]`.
- Switching between unary and binary operations must update the visible or
  enabled inputs and clear any stale second-operand validation error.

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

#### Behavioral testing contract

Define a lightweight public calculator abstraction:

```go
type Calculator interface {
    Calculate(operation string, operands []float64) (float64, error)
}

func NewCalculator() Calculator
```

`NewCalculator` must return an implementation of `Calculator` configured with
all supported operations. Behavioral tests must depend only on this interface
and must not access concrete operation structs, internal fields, the operation
registry, or other implementation details.

Required test coverage:
- success cases for every supported operation:
  - addition
  - subtraction
  - multiplication
  - division
  - exponentiation (`power`)
  - square root (`sqrt`)
  - percentage (`percent`)
- division by zero rejection
- negative square-root input rejection
- power behavior must match Go's `math.Pow`, including fractional and negative
  exponents where the result is finite
- power cases that produce a non-finite result must return
  `NON_FINITE_RESULT` with HTTP `422` through the API
- invalid operation rejection
- invalid payload rejection
- missing `operands` rejection
- null operands and null items rejection
- operand-array arity mismatch rejection for unary and binary operations
- health endpoint response

Recommended structure:
- behavioral tests through `Calculator`, including valid and
  operation-specific invalid inputs
- handler-level tests for HTTP responses and status codes
- validation tests for payload rules

Tests for the HTTP handler may verify the public JSON API contract, but the
calculator behavior tests must remain independent of the concrete operation
implementations and registry structure.

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
- verify addition, subtraction, multiplication, division, exponentiation,
  percentage, and square root work end-to-end
- verify the unary square-root operation shows only one operand input
- verify binary operations show both operand inputs
- verify invalid inputs are blocked
- verify division by zero returns a clean error
- verify a negative square-root input returns a clean error

## 10. Acceptance Criteria

The project is considered complete when all of the following are true:

1. The frontend loads and shows a calculator interface.
2. A user can compute addition, subtraction, multiplication, division,
   exponentiation, percentage, and square root.
3. The frontend sends requests to the backend and displays results.
4. Invalid inputs are blocked and user-friendly messages are shown.
5. Division by zero is rejected by the backend with a structured error response.
6. Negative square-root input is rejected by the backend with a structured error
   response.
7. Power results match Go's `math.Pow` semantics, and non-finite power results
   are returned as structured `NON_FINITE_RESULT` errors rather than JSON
   numbers.
8. The backend exposes a REST JSON API with predictable request/response
   contracts.
9. Unit tests cover every supported operation and validation path.
10. Setup instructions are documented in the project README.
11. The codebase is organized cleanly enough for another developer to continue
    without confusion.

## 11. Design Decisions and Assumptions

### 11.1 Why a single calculation endpoint?

A single endpoint reduces implementation time and keeps the API easy to reason about. It also makes testing simpler and reduces unnecessary complexity for a small assignment.

### 11.2 Why minimal dependencies?

The project should stay lean and fast to implement. Using a minimal Go stack and a simple React UI avoids dependency overhead and keeps the solution easy to maintain.

### 11.3 Why strict validation?

The assignment explicitly calls out invalid inputs and edge cases. Strong validation ensures the API is predictable and makes the frontend/backend contract easier to test.

### 11.4 Assumption: all specified operations are required

The backend and frontend must implement `add`, `subtract`, `multiply`, `divide`,
`power`, `sqrt`, and `percent`. These operations share the same API contract and
must be covered by validation and automated tests.

### 11.5 Assumption: no persistence layer

The assignment does not require storing calculation history or user data, and that is intentionally excluded from the minimum viable scope.

## 12. Implementation Notes for Future Agents

- Build the backend first, then wire the frontend to the API.
- Keep the API contract stable while implementing the UI.
- Prefer correctness and readable code over extra features.
- Do not expand scope beyond the specified operations unless time remains.
- All specified operations should be implemented as operation strings in the same
  request contract rather than as separate endpoint complexity.

## 13. Final Scope Recommendation

The recommended implementation is:
- React frontend with a simple calculator UI
- Go backend with a single `/api/calculate` endpoint
- required operations: add, subtract, multiply, divide, power, sqrt, percent
- validation for all edge cases and malformed inputs
- tests for backend logic and frontend smoke behavior
- documentation for setup and usage

This is the highest-confidence path to a clean, completed project within the designated time budget.

