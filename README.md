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

### Error response

Division by zero returns HTTP `422`:

```json
{
  "error": {
    "code": "DIVISION_BY_ZERO",
    "message": "cannot divide by zero"
  }
}
```

## Design decisions or assumptions

- A single `POST /api/calculate` endpoint handles `add`, `subtract`,
  `multiply`, `divide`, `power`, `sqrt`, and `percent`.
- Binary operations require two operands. `sqrt` requires one operand.
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
| Backend | 85.6% | not reported by `go tool cover` | not reported by `go tool cover` |
| Frontend | 91.3% | 93.65% | 87.5% |

## Repository

The project is maintained in the associated GitHub repository. The `prompts/`
directory contains the prompts and supporting artifacts used during
development.
