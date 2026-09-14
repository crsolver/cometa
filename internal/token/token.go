package token

import "hacha/internal/ast"

type Kind string

const (
	EOF     Kind = "EOF"
	Newline Kind = "NEWLINE"
	Indent  Kind = "INDENT"
	Dedent  Kind = "DEDENT"
	Ident   Kind = "IDENT"
	Number  Kind = "NUMBER"
	String  Kind = "STRING"

	Tipo      Kind = "tipo"
	Enum      Kind = "enum"
	Casos     Kind = "casos"
	Arrow     Kind = "=>"
	Fn        Kind = "fn"
	Var       Kind = "var"
	Si        Kind = "si"
	Osi       Kind = "osi"
	Sino      Kind = "sino"
	Repetir   Kind = "repetir"
	Continuar Kind = "continuar"
	Romper    Kind = "romper"
	Num       Kind = "num"
	Cadena    Kind = "cadena"
	Bool      Kind = "bool"
	True      Kind = "verdadero"
	False     Kind = "falso"

	LParen    Kind = "("
	RParen    Kind = ")"
	LBracket  Kind = "["
	RBracket  Kind = "]"
	LBrace    Kind = "{"
	RBrace    Kind = "}"
	Comma     Kind = ","
	Pipe      Kind = "|"
	Colon     Kind = ":"
	Dot       Kind = "."
	Range     Kind = ".."
	Ellipsis  Kind = "..."
	At        Kind = "@"
	Assign    Kind = "="
	Plus      Kind = "+"
	Minus     Kind = "-"
	Star      Kind = "*"
	Slash     Kind = "/"
	Percent   Kind = "%"
	Equal     Kind = "=="
	NotEqual  Kind = "!="
	Less      Kind = "<"
	LessEq    Kind = "<="
	Greater   Kind = ">"
	GreaterEq Kind = ">="
	And       Kind = "&&"
	Or        Kind = "||"
	Bang      Kind = "!"
	Question  Kind = "?"
	Fallback  Kind = "o"
	Catch     Kind = "capturar"
	Try       Kind = "intentar"
	Return    Kind = "retornar"
)

type Token struct {
	Kind   Kind
	Lexeme string
	Pos    ast.Pos
}
