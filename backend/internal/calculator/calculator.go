package calculator

import (
	"errors"
	"math"
)

const (
	CodeInvalidOperation = "INVALID_OPERATION"
	CodeInvalidArity     = "INVALID_ARITY"
	CodeDivisionByZero   = "DIVISION_BY_ZERO"
	CodeNegativeInput    = "NEGATIVE_INPUT"
	CodeNonFiniteResult  = "NON_FINITE_RESULT"
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

func operationRegistry() map[string]Operation {
	operations := []Operation{
		binaryOperation{"add", func(a, b float64) float64 { return a + b }},
		binaryOperation{"subtract", func(a, b float64) float64 { return a - b }},
		binaryOperation{"multiply", func(a, b float64) float64 { return a * b }},
		divideOperation{},
		binaryOperation{"power", func(a, b float64) float64 { return math.Pow(a, b) }},
		sqrtOperation{},
		binaryOperation{"percent", func(a, b float64) float64 { return a * b / 100 }},
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
		return 0, &Error{Code: CodeInvalidOperation, Message: "unsupported operation"}
	}
	if len(operands) != operation.Arity() {
		return 0, &Error{Code: CodeInvalidArity, Message: "operation requires a different number of operands"}
	}
	result, err := operation.Evaluate(operands)
	if err != nil {
		return 0, err
	}
	if !isFinite(result) {
		return 0, &Error{Code: CodeNonFiniteResult, Message: "calculation produced a non-finite result"}
	}
	return result, nil
}

type binaryOperation struct {
	name string
	fn   func(float64, float64) float64
}

func (o binaryOperation) Name() string { return o.name }
func (o binaryOperation) Arity() int   { return 2 }
func (o binaryOperation) Evaluate(operands []float64) (float64, error) {
	if len(operands) != o.Arity() {
		return 0, &Error{Code: CodeInvalidArity, Message: "operation requires two operands"}
	}
	return o.fn(operands[0], operands[1]), nil
}

type divideOperation struct{}

func (divideOperation) Name() string { return "divide" }
func (divideOperation) Arity() int   { return 2 }
func (divideOperation) Evaluate(operands []float64) (float64, error) {
	if len(operands) != 2 {
		return 0, &Error{Code: CodeInvalidArity, Message: "operation requires two operands"}
	}
	if operands[1] == 0 {
		return 0, &Error{Code: CodeDivisionByZero, Message: "cannot divide by zero"}
	}
	return operands[0] / operands[1], nil
}

type sqrtOperation struct{}

func (sqrtOperation) Name() string { return "sqrt" }
func (sqrtOperation) Arity() int   { return 1 }
func (sqrtOperation) Evaluate(operands []float64) (float64, error) {
	if len(operands) != 1 {
		return 0, &Error{Code: CodeInvalidArity, Message: "operation requires one operand"}
	}
	if operands[0] < 0 {
		return 0, &Error{Code: CodeNegativeInput, Message: "square root input cannot be negative"}
	}
	return math.Sqrt(operands[0]), nil
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
