# Calculator

Full-stack calculator application with a React and TypeScript frontend and a
Go REST backend.

## Setup instructions

### Prerequisites

- Go 1.20 or later
- Node.js 22 or later and npm
- Docker Desktop, optional for the containerized setup

Install frontend dependencies:

```powershell
Set-Location frontend
npm install
```

The backend has no third-party dependencies. Go dependencies are resolved by
the Go toolchain from `backend/go.mod`.

## How to run the frontend and backend

### Run the backend locally

In one terminal:

```powershell
Set-Location backend
go run .\cmd\server
```

The backend listens on `http://localhost:8080`. Set `ADDR` to use another
address, for example `$env:ADDR = ":9090"`.

### Run the frontend locally

In a second terminal:

```powershell
Set-Location frontend
npm run dev
```

Open the Vite URL shown in the terminal, normally
`http://localhost:5173`. The Vite development proxy forwards `/api` requests
to the backend on port 8080.

### Run with Docker

If you are on Windows and the container exits immediately, make sure the
`docker/entrypoint.sh` file uses Unix line endings before building the image:

```powershell
(Get-Content .\docker\entrypoint.sh -Raw) -replace "`r`n","`n" | Set-Content .\docker\entrypoint.sh -NoNewline
```

Build the production image from the repository root:

```powershell
docker build -t calculator .
```

Run the application:

```powershell
docker run --rm -p 8080:80 calculator
```

Open `http://localhost:8080`. nginx serves the compiled frontend and proxies
API requests to the Go backend inside the same container.

## Examples of API calls

The backend exposes a JSON REST API.

### Health check

```powershell
Invoke-RestMethod http://localhost:8080/api/health
```

Response:

```json
{
  "status": "ok"
}
```

### Addition

```powershell
Invoke-RestMethod `
  -Uri http://localhost:8080/api/calculate `
  -Method Post `
  -ContentType "application/json" `
  -Body '{"operation":"add","operands":[12,3]}'
```

Response:

```json
{
  "result": 15
}
```

### Square root

```powershell
Invoke-RestMethod `
  -Uri http://localhost:8080/api/calculate `
  -Method Post `
  -ContentType "application/json" `
  -Body '{"operation":"sqrt","operands":[81]}'
```

Response:

```json
{
  "result": 9
}
```

### Error examples

The API returns structured JSON errors with the HTTP status and an `error.code`
field. The examples below show the most common validation failures.

#### Division by zero

```powershell
try {
  Invoke-RestMethod `
    -Uri http://localhost:8080/api/calculate `
    -Method Post `
    -ContentType "application/json" `
    -Body '{"operation":"divide","operands":[10,0]}'
} catch {
  $_.ErrorDetails.Message
}
```

Response:

```json
{
  "error": {
    "code": "DIVISION_BY_ZERO",
    "message": "cannot divide by zero"
  }
}
```

#### Invalid operation

```powershell
try {
  Invoke-RestMethod `
    -Uri http://localhost:8080/api/calculate `
    -Method Post `
    -ContentType "application/json" `
    -Body '{"operation":"mod","operands":[10,3]}'
} catch {
  $_.ErrorDetails.Message
}
```

Response:

```json
{
  "error": {
    "code": "INVALID_OPERATION",
    "message": "Operation must be one of: add, subtract, multiply, divide, power, sqrt, percent"
  }
}
```

#### Missing operand

```powershell
try {
  Invoke-RestMethod `
    -Uri http://localhost:8080/api/calculate `
    -Method Post `
    -ContentType "application/json" `
    -Body '{"operation":"add","operands":[12]}'
} catch {
  $_.ErrorDetails.Message
}
```

Response:

```json
{
  "error": {
    "code": "MISSING_FIELD",
    "message": "a required operand is missing"
  }
}
```

#### Square root of a negative number

```powershell
try {
  Invoke-RestMethod `
    -Uri http://localhost:8080/api/calculate `
    -Method Post `
    -ContentType "application/json" `
    -Body '{"operation":"sqrt","operands":[-1]}'
} catch {
  $_.ErrorDetails.Message
}
```

Response:

```json
{
  "error": {
    "code": "NEGATIVE_INPUT",
    "message": "square root input cannot be negative"
  }
}
```

#### Non-finite calculation result

```powershell
try {
  Invoke-RestMethod `
    -Uri http://localhost:8080/api/calculate `
    -Method Post `
    -ContentType "application/json" `
    -Body '{"operation":"power","operands":[0,-1]}'
} catch {
  $_.ErrorDetails.Message
}
```

Response:

```json
{
  "error": {
    "code": "NON_FINITE_RESULT",
    "message": "calculation produced a non-finite result"
  }
}
```

## Design decisions or assumptions

- A single `POST /api/calculate` endpoint was chosen instead of separate
  endpoints for each operation because it was redundant: all calculator
  operations share the same request contract, validation flow, and response
  structure. Using one endpoint keeps the API consistent and avoids code
  duplication.
- A single microservice was preferred over a microservices architecture. Even
  though each operation could have been split into separate services, that
  would have been over-engineering for a tiny calculator. The project keeps a
  single service with clear internal boundaries rather than introducing
  unnecessary network and operational complexity.
- The different calculator operations are implemented through interfaces in a
  strategy-like pattern. Each operation exposes a common contract (`Name`,
  `Arity`, and `Evaluate`) and is registered centrally. This keeps the logic
  open for extension by registering new operations without changing the rest of
  the code. The code remains closed to modification in the core flow while
  allowing easy extension through the registry.
- Arithmetic and domain validation live in the Go backend. The frontend
  performs input validation and presentation formatting but does not duplicate
  calculation logic.
- Requests use an `operands` array so unary and binary operations share one
  API shape.
- Missing fields and explicit `null` values are distinguished during JSON
  decoding and return different structured error codes.
- The backend rejects non-finite operands and results so JSON responses never
  contain `NaN` or infinity.
- The backend returns finite raw `float64` values. The frontend formats large
  and small values for readability.
- The standard-library `net/http` package keeps the backend dependency-light.
- The Docker image uses multi-stage builds, nginx for static files and API
  proxying, and a Go process for the calculator service.
- Support for complex mathematical expressions such as `2 + 3 * 4` or multi-step
  formulas was intentionally not included. Implementing that would require a
  parser and execution engine, likely using patterns such as the interpreter
  pattern, which falls outside the scope of this small implementation and the
  project’s time budget.

## Spec-driven development approach

The project was implemented using a spec-driven development approach. The
core effort was focused on defining a clear specification document first so
that AI agents could implement the solution in a way that was easy to validate
and grounded in solid acceptance criteria, without writing implementation code
up front.

This meant that most of the work was invested in designing a precise
specification, rather than directly coding the full behavior from scratch. The
spec captured the functional requirements, expected API contract, validation
rules, and error behavior, allowing the implementation and testing work to be
performed with more confidence and less ambiguity.

In a second step, a small portion of code was added only to enable a testing
agent and an implementation agent to work in parallel. That created a light
TDD-like workflow: the tests and the implementation could evolve together while
still being anchored to the same specification. The goal was not to overbuild
or over-engineer the solution, but to create a structure where validation was
possible early and iteration remained fast.

Because the specification was well defined, less time was spent on direct
implementation. Most corrections were handled through small, targeted prompts
and iterative refinements, instead of broad rewrites. Once the behavior was
stable, the final documentation and the other project deliverables were written
and aligned with the validated implementation.

## Validation and test coverage

Run the backend tests and static checks:

```powershell
Set-Location backend
go test ./...
go vet ./...
```

Run frontend tests, build, and lint:

```powershell
Set-Location frontend
npm test -- --run
npm run build
npm run lint
```

Generate the backend coverage profile and summary:

```powershell
Set-Location backend
go test ./... -coverprofile .\coverage.out
go tool cover -func .\coverage.out
go tool cover -html .\coverage.out -o .\coverage.html
```

Generate the frontend coverage summary and HTML report:

```powershell
Set-Location frontend
npm run test:coverage
```

The backend HTML report is written to `backend/coverage.html`. The frontend
HTML report is written to `frontend/coverage/index.html`. Generated reports
are ignored by Git.

Latest local results:

| Suite | Statements | Lines | Branches |
| --- | ---: | ---: | ---: |
| Backend total (includes `cmd/server`) | 85.6% | not reported by `go tool cover` | not reported by `go tool cover` |
| Backend core package (`internal/calculator`) | 89.1% | not reported by `go tool cover` | not reported by `go tool cover` |
| Frontend | 91.3% | 93.65% | 87.5% |

> The backend overall coverage is 85.6% because it includes the small `cmd/server`
> package, which has no direct test coverage. The calculation logic itself in
> `internal/calculator` is covered at 89.1%.

## Repository

The project is maintained in the associated GitHub repository. The `prompts/`
directory contains the prompts and supporting artifacts used during
development.
