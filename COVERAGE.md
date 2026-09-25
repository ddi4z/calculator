# Test Coverage

## Regenerate the reports

Run the backend coverage commands from the repository root:

```powershell
Set-Location backend
go test ./... -coverprofile .\coverage.out
go tool cover -func .\coverage.out
go tool cover -html .\coverage.out -o .\coverage.html
```

The function summary is printed in the terminal. The HTML report is written to
`backend/coverage.html`.

Run the frontend coverage command:

```powershell
Set-Location frontend
npm run test:coverage
```

This prints the frontend coverage summary and writes the HTML report to
`frontend/coverage/index.html`. Run the backend and frontend commands from
separate terminals, or return to the repository root between them. The generated
coverage reports are ignored by Git.

## Latest recorded results

| Suite | Statements | Lines | Branches |
| --- | ---: | ---: | ---: |
| Backend total (includes `cmd/server`) | 85.9% | not reported by `go tool cover` | not reported by `go tool cover` |
| Backend core package (`internal/calculator`) | 89.3% | not reported by `go tool cover` | not reported by `go tool cover` |
| Frontend | 91.3% | 91.04% | 87.5% |

The backend overall coverage is 85.9% because it includes the small
`cmd/server` package, which has no direct test coverage. The calculator logic
in `internal/calculator` is covered at 89.3%.