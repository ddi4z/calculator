import { useMemo, useState } from 'react'
import './App.css'

type Operation =
  | 'add'
  | 'subtract'
  | 'multiply'
  | 'divide'
  | 'power'
  | 'sqrt'
  | 'percent'

type ApiError = {
  error?: {
    code?: string
    message?: string
  }
}

const operations: { value: Operation; label: string; symbol: string }[] = [
  { value: 'add', label: 'Addition', symbol: '+' },
  { value: 'subtract', label: 'Subtraction', symbol: '−' },
  { value: 'multiply', label: 'Multiplication', symbol: '×' },
  { value: 'divide', label: 'Division', symbol: '÷' },
  { value: 'power', label: 'Exponentiation', symbol: '^' },
  { value: 'sqrt', label: 'Square root', symbol: '√' },
  { value: 'percent', label: 'Percentage', symbol: '%' },
]

const binaryOperations = new Set<Operation>([
  'add',
  'subtract',
  'multiply',
  'divide',
  'power',
  'percent',
])

function formatResult(result: number) {
  return Number.isInteger(result)
    ? result.toString()
    : result.toLocaleString('en-US', { maximumFractionDigits: 10 })
}

function App() {
  const [operation, setOperation] = useState<Operation>('add')
  const [first, setFirst] = useState('')
  const [second, setSecond] = useState('')
  const [result, setResult] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const isBinary = binaryOperations.has(operation)
  const validationError = useMemo(() => {
    if (first.trim() === '') return 'Enter a value for the first number.'
    if (!Number.isFinite(Number(first))) return 'The first number must be valid.'
    if (operation === 'sqrt' && Number(first) < 0) {
      return 'Square root requires a non-negative number.'
    }
    if (isBinary) {
      if (second.trim() === '') return 'Enter a value for the second number.'
      if (!Number.isFinite(Number(second))) return 'The second number must be valid.'
      if (operation === 'divide' && Number(second) === 0) {
        return 'The divisor cannot be zero.'
      }
    }
    return null
  }, [first, second, operation, isBinary])

  const handleOperationChange = (nextOperation: Operation) => {
    setOperation(nextOperation)
    setError(null)
    setResult(null)
    if (!binaryOperations.has(nextOperation)) setSecond('')
  }

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (validationError) {
      setError(validationError)
      setResult(null)
      return
    }

    const operands = isBinary ? [Number(first), Number(second)] : [Number(first)]
    setError(null)
    setResult(null)
    setIsLoading(true)

    try {
      const response = await fetch('/api/calculate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ operation, operands }),
      })
      const payload = (await response.json()) as { result?: number } & ApiError
      if (!response.ok) {
        throw new Error(payload.error?.message || 'The calculation could not be completed.')
      }
      if (typeof payload.result !== 'number' || !Number.isFinite(payload.result)) {
        throw new Error('The backend returned an invalid result.')
      }
      setResult(formatResult(payload.result))
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : 'Unable to reach the calculator service.',
      )
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <main className="calculator-shell">
      <section className="calculator-card" aria-labelledby="calculator-title">
        <div className="eyebrow">SEZZLE CALCULATOR</div>
        <h1 id="calculator-title">Make a calculation</h1>
        <p className="intro">Choose an operation and enter your numbers below.</p>

        <form onSubmit={handleSubmit} noValidate>
          <label className="field-label" htmlFor="operation">
            Operation
          </label>
          <select
            id="operation"
            value={operation}
            onChange={(event) => handleOperationChange(event.target.value as Operation)}
          >
            {operations.map((item) => (
              <option key={item.value} value={item.value}>
                {item.symbol} {item.label}
              </option>
            ))}
          </select>

          <div className={`operand-grid ${isBinary ? '' : 'unary'}`}>
            <div className="operand-field">
              <label className="field-label" htmlFor="first-number">
                {isBinary ? 'First number' : 'Number'}
              </label>
              <input
                id="first-number"
                type="number"
                inputMode="decimal"
                value={first}
                onChange={(event) => {
                  setFirst(event.target.value)
                  setError(null)
                  setResult(null)
                }}
                aria-invalid={Boolean(error && validationError)}
                placeholder="e.g. 12"
              />
            </div>
            {isBinary && (
              <div className="operand-field">
                <label className="field-label" htmlFor="second-number">
                  Second number
                </label>
                <input
                  id="second-number"
                  type="number"
                  inputMode="decimal"
                  value={second}
                  onChange={(event) => {
                    setSecond(event.target.value)
                    setError(null)
                    setResult(null)
                  }}
                  aria-invalid={Boolean(error && validationError)}
                  placeholder="e.g. 3"
                />
              </div>
            )}
          </div>

          {validationError && <p className="inline-error">{validationError}</p>}

          <button type="submit" disabled={Boolean(validationError) || isLoading}>
            {isLoading ? 'Calculating…' : 'Calculate'}
          </button>
        </form>

        <div className="feedback" aria-live="polite">
          {result !== null && (
            <div className="result-panel">
              <span className="feedback-label">Result</span>
              <strong>{result}</strong>
            </div>
          )}
          {error && !validationError && (
            <p className="api-error" role="alert">
              {error}
            </p>
          )}
        </div>
      </section>
    </main>
  )
}

export default App
