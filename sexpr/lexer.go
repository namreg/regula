package sexpr

type Token int

const (
	EOF Token = iota
	WHITESPACE
	LPAREN // (
	RPAREN // )
	STRING
	NUMBER
	BOOL
	COMMENT
	SYMBOL
)

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r' || r == '\n'
}

func isLParen(r rune) bool {
	return r == '('
}

func isRParen(r rune) bool {
	return r == ')'
}

func isString(r rune) bool {
	return r == '"'
}
