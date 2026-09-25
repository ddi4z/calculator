package calculator

import (
	"errors"
	"math"
)

const (
	CodeInvalidOperation = "INVALID_OPERATION"
	CodeInvalidArity     = "INVALID_ARITY"
	CodeInvalidNumber    = "INVALID_NUMBER"
	CodeDivisionByZero   = "DIVISION_BY_ZERO"
	CodeNegativeInput    = "NEGATIVE_INPUT"
	CodeNonFiniteResult  = "NON_FINITE_RESULT"

	MessageInvalidOperation = "Operation must be one of: add, subtract, multiply, divide, power, sqrt, percent"
	MessageInvalidArity     = "operation requires a different number of operands"
	MessageInvalidArityOne  = "operation requires one operand"
	MessageInvalidArityTwo  = "operation requires two operands"
	MessageInvalidNumber    = "operand must be a finite number"
	MessageDivisionByZero   = "cannot divide by zero"
	MessageNegativeInput    = "square root input cannot be negative"
	MessageNonFiniteResult  = "calculation produced a non-finite result"
)

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

type Calculator interface {
	Calculate(operation string, operands []float64) (float64, error)
}

type Operation interface {
	Name() string
	Arity() int
	Evaluate(operands []float64) (float64, error)
}

type calculator struct {
	operations map[string]Operation
}

func NewCalculator() Calculator {
	return &calculator{operations: operationRegistry()}
}

var operationCatalog = buildOperationRegistry()

func operationRegistry() map[string]Operation {
	return operationCatalog
}

func buildOperationRegistry() map[string]Operation {
	operations := []Operation{
		binaryOperation{name: "add", fn: func(a, b float64) (float64, error) { return a + b, nil }},
		binaryOperation{name: "subtract", fn: func(a, b float64) (float64, error) { return a - b, nil }},
		binaryOperation{name: "multiply", fn: func(a, b float64) (float64, error) { return a * b, nil }},
		binaryOperation{
			name: "divide",
			fn: func(a, b float64) (float64, error) {
				if b == 0 {
					return 0, &Error{Code: CodeDivisionByZero, Message: MessageDivisionByZero}
				}
				return a / b, nil
			},
		},
		binaryOperation{name: "power", fn: func(a, b float64) (float64, error) { return math.Pow(a, b), nil }},
		unaryOperation{
			name: "sqrt",
			fn: func(value float64) (float64, error) {
				if value < 0 {
					return 0, &Error{Code: CodeNegativeInput, Message: MessageNegativeInput}
				}
				return math.Sqrt(value), nil
			},
		},
		binaryOperation{name: "percent", fn: func(a, b float64) (float64, error) { return a * b / 100, nil }},
	}
	registry := make(map[string]Operation, len(operations))
	for _, operation := range operations {
		registry[operation.Name()] = operation
	}
	return registry
}

func (c *calculator) Calculate(name string, operands []float64) (float64, error) {
	operation, ok := c.operations[name]
	if !ok {
		return 0, &Error{Code: CodeInvalidOperation, Message: MessageInvalidOperation}
	}
	if len(operands) != operation.Arity() {
		return 0, &Error{Code: CodeInvalidArity, Message: MessageInvalidArity}
	}
	for _, operand := range operands {
		if !isFinite(operand) {
			return 0, &Error{Code: CodeInvalidNumber, Message: MessageInvalidNumber}
		}
	}
	result, err := operation.Evaluate(operands)
	if err != nil {
		return 0, err
	}
	if !isFinite(result) {
		return 0, &Error{Code: CodeNonFiniteResult, Message: MessageNonFiniteResult}
	}
	return result, nil
}

type binaryOperation struct {
	name string
	fn   func(float64, float64) (float64, error)
}

func (o binaryOperation) Name() string { return o.name }
func (o binaryOperation) Arity() int   { return 2 }
func (o binaryOperation) Evaluate(operands []float64) (float64, error) {
	if len(operands) != o.Arity() {
		return 0, &Error{Code: CodeInvalidArity, Message: MessageInvalidArityTwo}
	}
	return o.fn(operands[0], operands[1])
}

type unaryOperation struct {
	name string
	fn   func(float64) (float64, error)
}

func (o unaryOperation) Name() string { return o.name }
func (o unaryOperation) Arity() int   { return 1 }
func (o unaryOperation) Evaluate(operands []float64) (float64, error) {
	if len(operands) != 1 {
		return 0, &Error{Code: CodeInvalidArity, Message: MessageInvalidArityOne}
	}
	return o.fn(operands[0])

}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func ErrorCode(err error) string {
	var calculationError *Error
	if errors.As(err, &calculationError) {
		return calculationError.Code
	}
	return ""
}
