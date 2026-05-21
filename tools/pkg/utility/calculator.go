// Package utility provides utility tools.
package utility

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/oneliang/aura/shared/pkg/constants"
	tools "github.com/oneliang/aura/tools/pkg"
)

// CalculatorTool performs mathematical calculations.
type CalculatorTool struct{}

// NewCalculatorTool creates a new calculator tool.
func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{}
}

// Name returns the tool name.
func (t *CalculatorTool) Name() string {
	return constants.ToolCalculator
}

// Description returns the tool description.
func (t *CalculatorTool) Description() string {
	return "Perform mathematical calculations. Parameters: expression (string, required) - supports basic math: +, -, *, /, ^, ( ), and functions: sqrt, sin, cos, tan, log, abs"
}

// Execute evaluates a mathematical expression.
func (t *CalculatorTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	expression, ok := params["expression"].(string)
	if !ok {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "expression parameter is required"}, nil
	}

	result, err := evaluate(expression)
	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: err.Error()}, nil
	}

	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: fmt.Sprintf("Result: %g", result),
	}, nil
}

func evaluate(expr string) (float64, error) {
	expr = strings.ToLower(strings.TrimSpace(expr))

	// Replace common constants and operators
	replacer := strings.NewReplacer(
		"sqrt", "Q",
		"sin", "S",
		"cos", "c",
		"tan", "t",
		"abs", "a",
		"log", "l",
		"pi", fmt.Sprintf("%g", math.Pi),
		"e", fmt.Sprintf("%g", math.E),
		"^", "**",
	)
	expr = replacer.Replace(expr)

	// Remove all spaces
	expr = strings.ReplaceAll(expr, " ", "")

	// Tokenize and evaluate
	tokens := tokenize(expr)
	if len(tokens) == 0 {
		return 0, fmt.Errorf("empty expression")
	}

	result, err := evalTokens(tokens)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func tokenize(expr string) []string {
	var tokens []string
	var current strings.Builder

	for i, r := range expr {
		if unicode.IsDigit(r) || r == '.' {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			if !unicode.IsSpace(r) {
				tokens = append(tokens, string(r))
			}
		}

		// Handle unary minus
		if r == '-' && (i == 0 || expr[i-1] == '(' || expr[i-1] == '+' || expr[i-1] == '-' || expr[i-1] == '*' || expr[i-1] == '/') {
			if current.Len() == 0 {
				current.WriteRune(r)
			}
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

func evalTokens(tokens []string) (float64, error) {
	// Use recursive descent parsing
	parser := &exprParser{tokens: tokens}
	return parser.parseExpression()
}

type exprParser struct {
	tokens []string
	pos    int
}

func (p *exprParser) parseExpression() (float64, error) {
	return p.parseAddSub()
}

func (p *exprParser) parseAddSub() (float64, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return 0, err
	}

	for p.pos < len(p.tokens) {
		token := p.tokens[p.pos]
		if token == "+" || token == "-" {
			p.pos++
			right, err := p.parseMulDiv()
			if err != nil {
				return 0, err
			}
			if token == "+" {
				left += right
			} else {
				left -= right
			}
		} else {
			break
		}
	}

	return left, nil
}

func (p *exprParser) parseMulDiv() (float64, error) {
	left, err := p.parsePower()
	if err != nil {
		return 0, err
	}

	for p.pos < len(p.tokens) {
		token := p.tokens[p.pos]
		if token == "*" || token == "/" {
			p.pos++
			right, err := p.parsePower()
			if err != nil {
				return 0, err
			}
			if token == "*" {
				left *= right
			} else {
				if right == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				left /= right
			}
		} else {
			break
		}
	}

	return left, nil
}

func (p *exprParser) parsePower() (float64, error) {
	left, err := p.parseUnary()
	if err != nil {
		return 0, err
	}

	if p.pos < len(p.tokens) && p.tokens[p.pos] == "**" {
		p.pos++
		right, err := p.parsePower()
		if err != nil {
			return 0, err
		}
		return math.Pow(left, right), nil
	}

	return left, nil
}

func (p *exprParser) parseUnary() (float64, error) {
	if p.pos < len(p.tokens) && p.tokens[p.pos] == "-" {
		p.pos++
		val, err := p.parsePrimary()
		if err != nil {
			return 0, err
		}
		return -val, nil
	}

	if p.pos < len(p.tokens) && p.tokens[p.pos] == "+" {
		p.pos++
		return p.parsePrimary()
	}

	return p.parsePrimary()
}

func (p *exprParser) parsePrimary() (float64, error) {
	if p.pos >= len(p.tokens) {
		return 0, fmt.Errorf("unexpected end of expression")
	}

	token := p.tokens[p.pos]

	// Number
	if val, err := strconv.ParseFloat(token, 64); err == nil {
		p.pos++
		return val, nil
	}

	// Parentheses
	if token == "(" {
		p.pos++
		expr, err := p.parseExpression()
		if err != nil {
			return 0, err
		}
		if p.pos >= len(p.tokens) || p.tokens[p.pos] != ")" {
			return 0, fmt.Errorf("unmatched parentheses")
		}
		p.pos++
		return expr, nil
	}

	// Functions
	if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1] == "(" {
		funcName := token
		p.pos += 2 // Skip function name and (
		arg, err := p.parseExpression()
		if err != nil {
			return 0, err
		}
		if p.pos >= len(p.tokens) || p.tokens[p.pos] != ")" {
			return 0, fmt.Errorf("unmatched parentheses in function call")
		}
		p.pos++

		return applyFunction(funcName, arg)
	}

	return 0, fmt.Errorf("unexpected token: %s", token)
}

func applyFunction(name string, arg float64) (float64, error) {
	switch name {
	case "Q":
		return math.Sqrt(arg), nil
	case "S":
		return math.Sin(arg), nil
	case "cos":
		return math.Cos(arg), nil
	case "tan":
		return math.Tan(arg), nil
	case "log", "l":
		return math.Log10(arg), nil
	case "ln":
		return math.Log(arg), nil
	case "abs", "a":
		return math.Abs(arg), nil
	default:
		return 0, fmt.Errorf("unknown function: %s", name)
	}
}
