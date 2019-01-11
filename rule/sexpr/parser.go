package sexpr

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/heetch/regula/rule"
)

// makeSymbolMap returns a opCodeMap with the full map of the built-in
// symbols of our symbolic expression language to their implementation
// as regula rule.Operators.
func makeSymbolMap() *opCodeMap {
	sm := newOpCodeMap()
	sm.mapSymbol("=", "eq")
	sm.mapSymbol("+", "add")
	sm.mapSymbol("-", "sub")
	sm.mapSymbol("*", "mult")
	sm.mapSymbol("/", "div")
	sm.mapSymbol("%", "mod")
	sm.mapSymbol("in", "in")
	sm.mapSymbol("and", "and")
	sm.mapSymbol("or", "or")
	sm.mapSymbol("not", "not")
	sm.mapSymbol(">", "gt")
	sm.mapSymbol(">=", "gte")
	sm.mapSymbol("<", "lt")
	sm.mapSymbol("<=", "lte")
	sm.mapSymbol("hash", "hash")
	sm.mapSymbol("percentile", "percentile")
	sm.mapSymbol("int->float", "intToFloat")
	return sm
}

type Parameters map[string]rule.Type

// Parser is a parser for the Regula Symbolic Expression Language.  It
// should always be constructed by passing an io.Reader to the
// NewParser method.
type Parser struct {
	s         *Scanner
	buf       *lexicalElement
	buffered  bool
	opCodeMap *opCodeMap
}

// NewParser returns a new Parser instance that can be used to parse a
// tree of symbolic expressions from the provide io.Reader.  No
// parsing will occur until the resulting Parser's Parse method is
// called.
func NewParser(r io.Reader) *Parser {
	return &Parser{
		s:         NewScanner(r),
		opCodeMap: makeSymbolMap(),
	}
}

// scan returns the next lexicalElement from the text to be parsed, or
// the buffered element if unscan was called prior to scan.
func (p *Parser) scan() (*lexicalElement, error) {
	var err error
	if !p.buffered {
		var next *lexicalElement
		for {
			next, err = p.s.Scan()
			if err != nil {
				break
			}
			// Ignore white-space
			if next.Token != WHITESPACE {
				p.buf = next
				break
			}
		}

	}
	p.buffered = false
	return p.buf, err
}

// unscan instructs the Parser to use the buffered value for the next call to scan
func (p *Parser) unscan() {
	p.buffered = true
}

// Parse will parse the first expression it finds in a buffer,
// including all its sub-expressions.  If an error is encountered this
// will be returned, otherwise an abstract syntax tree built of Exprs
// will be returned.
func (p *Parser) Parse(params Parameters) (rule.Expr, error) {
	// Our root expression *must* have the return type BOOLEAN
	t := rule.Term{
		Type:        rule.BOOLEAN,
		Cardinality: rule.ONE,
	}
	expr, err := p.parseExpression(params)
	if err != nil {
		return nil, err
	}
	if !t.IsFulfilledBy(expr) {
		return nil, ParserError{
			Msg:             fmt.Sprintf("The root expression in a rule must return a Boolean, but it returns %s", expr.Contract().ReturnType),
			ErrorType:       TYPE,
			StartChar:       0,
			EndChar:         p.s.charCount,
			StartByte:       0,
			EndByte:         p.s.byteCount,
			StartLine:       0,
			EndLine:         p.s.lineCount,
			StartCharInLine: 0,
			EndCharInLine:   p.s.lineCharCount,
		}
	}

	// TODO: check that there are no further expression in the
	// buffer (other than whitespace and comments) as this would
	// be an error condition.
	return expr, nil
}

// parseExpression parses a single Expr from the Parsers buffer.  If
// an expression contains other expressions these will be parsed as
// they are encountered.  In this way parse expression will
// recursively walk a tree of expressions and the returned Expr will
// be the top of an abstract syntax tree representing the full tree of
// expression encompassed by the current text expression.  Any errors
// encountered will be passed back up the tree.
func (p *Parser) parseExpression(params Parameters) (rule.Expr, error) {
	var expr rule.Expr
	var inOperator bool
	var opExpr rule.Operator

Loop:
	for {
		le, err := p.scan()
		if err != nil {
			return nil, err
		}
		switch le.Token {
		case EOF:
			// The parser shouldn't actually hit this
			// case, because we should only ever parse a
			// single tree of expressions (terminated by a
			// right parenthesis), a single value or a
			// single param.  This will become more cloudy
			// if we start parsing comments.
			return nil, newParserError(le, fmt.Errorf("unexpected end of file"))
		case COMMENT:
			// We ignore comments, for now
			continue
		case WHITESPACE:
			// We just ignore white space, for now.
			continue
		case LPAREN:
			if inOperator {
				// This is a nested operator, so we'll
				// want to treat it exactly like we
				// did this one.  ... therefore, pop that left parenthesis back.
				p.unscan()
				// .... and recur!
				expr, err = p.parseExpression(params)
				if err != nil {
					return nil, err
				}
				// Now we've got our nested
				// expression, lets push it onto this
				// one (and make sure it complies with
				// the contract)
				if err := opExpr.PushExpr(expr); err != nil {
					// TODO: drastically improve this error message
					return nil, newParserError(le, fmt.Errorf(
						"Type mismatch in subexpression",
					))
				}
				continue
			}
			// Left parenthesis must be followed by an operator
			opExpr, err = p.parseOperator()
			if err != nil {
				return nil, err
			}
			inOperator = true

		case BOOL:
			expr = p.makeBoolValue(le)
			if !inOperator {
				// Great, we got a BoolValue, let's return that.
				break Loop
			}
			// OK, let's push the BoolValue into our
			// operator and see if that complies with the
			// contract.
			if err := opExpr.PushExpr(expr); err != nil {
				return nil, newParserError(le, err)
			}

		case STRING:
			expr = rule.StringValue(le.Literal)
			if !inOperator {
				// OK, we're done, break the loop
				break Loop
			}
			// Lets see if our opExpr is really expecting a string
			if err := opExpr.PushExpr(expr); err != nil {
				return nil, newParserError(le, err)
			}

		case NUMBER:
			expr, err := p.makeNumber(le)
			if err != nil {
				return nil, err
			}
			if !inOperator {
				// Just return the Number
				break Loop
			}
			if err := opExpr.PushExpr(expr); err != nil {
				return nil, newParserError(le, err)
			}

		case SYMBOL:
			expr, err = p.makeParameter(le, params)
			if err != nil {
				return nil, err
			}
			if !inOperator {
				break Loop
			}
			if err := opExpr.(rule.Operator).PushExpr(expr); err != nil {
				return nil, newParserError(le, err)
			}
		case RPAREN:
			if !inOperator {
				// We don't have a matching LPAREN, so this is bad news.
				return nil, newParserError(le, fmt.Errorf("Unexpected closing parenthesis"))
			}
			// We've close off the operator
			inOperator = false
			// .. lets finalise it, and see if that's compatible with the contract
			if err := opExpr.Finalise(); err != nil {
				return nil, newParserError(le, err)
			}
			// All is good, the output expression is this operater
			expr = opExpr
			break Loop
		}

	}
	return expr, nil
}

// makeBoolValue extracts a BoolValue from a lexicalElement, or returns an error
func (p *Parser) makeBoolValue(le *lexicalElement) rule.Expr {
	if le.Literal == "true" {
		return rule.BoolValue(true)
	}
	return rule.BoolValue(false)
}

// parseOperator attempts to convert the next symbol into an operator, if it fails and error is returned.
func (p *Parser) parseOperator() (rule.Operator, error) {
	le, err := p.scan()
	if err != nil {
		return nil, err
	}
	if le.Token != SYMBOL {
		return nil, fmt.Errorf("Expected an operator, but got the %s %q", le.Token, le.Literal)
	}

	op, err := p.opCodeMap.getOperatorForSymbol(le.Literal)
	if err != nil {
		return nil, err
	}

	return op, nil
}

//
func (p *Parser) makeNumber(le *lexicalElement) (rule.Expr, error) {
	if strings.ContainsRune(le.Literal, '.') {
		f64, err := strconv.ParseFloat(le.Literal, 64)
		if err != nil {
			return nil, newParserError(le, err)
		}
		return rule.Float64Value(f64), nil
	}
	i64, err := strconv.ParseInt(le.Literal, 10, 64)
	if err != nil {
		return nil, newParserError(le, err)
	}
	return rule.Int64Value(i64), nil
}

// makeParameter looks up a Symbol in the Parameters table and returns a Param of the correct type, or otherwise throws an error.
func (p *Parser) makeParameter(le *lexicalElement, params Parameters) (rule.Expr, error) {
	t, ok := params[le.Literal]
	if !ok {
		return nil, newParserError(le, fmt.Errorf(
			"unknown parameter name %q", le.Literal,
		))
	}
	switch t {
	case rule.BOOLEAN:
		return rule.BoolParam(le.Literal), nil
	case rule.STRING:
		return rule.StringParam(le.Literal), nil
	case rule.INTEGER:
		return rule.Int64Param(le.Literal), nil
	case rule.FLOAT:
		return rule.Float64Param(le.Literal), nil
	}
	// 🛈: NUMBER and ANY are not valid types for parameters
	return nil, newParserError(le, fmt.Errorf("parameter %q has an invalid Type: %s", le.Literal, t))
}