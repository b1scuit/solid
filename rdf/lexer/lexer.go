package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/b1scuit/solid/rdf/lexer/lexertoken"
)

type LexerOption func(*Lexer)

func WithName(name string) LexerOption {
	return func(l *Lexer) {
		l.Name = name
	}
}

func WithInput(i string) LexerOption {
	return func(l *Lexer) {
		l.Input = i
	}
}

type Lexer struct {
	Name   string
	Input  string
	Tokens chan lexertoken.Token
	State  LexFn

	Start int
	Pos   int
	Width int
}

func New(opts ...LexerOption) (*Lexer, error) {
	l := &Lexer{
		State:  LexBegin,
		Tokens: make(chan lexertoken.Token),
	}

	for _, f := range opts {
		f(l)
	}

	return l, nil
}

/*
Backup to the beginning of the last read token.
*/
func (this *Lexer) Backup() {
	this.Pos -= this.Width
}

/*
Returns a slice of the current input from the current lexer start position
to the current position.
*/
func (this *Lexer) CurrentInput() string {
	return this.Input[this.Start:this.Pos]
}

/*
Decrement the position
*/
func (this *Lexer) Dec() {
	this.Pos--
}

/*
Puts a token onto the token channel. The value of this token is
read from the input based on the current lexer position.
*/
func (this *Lexer) Emit(tokenType lexertoken.TokenType) {
	this.Tokens <- lexertoken.Token{Type: tokenType, Value: strings.TrimSpace(this.Input[this.Start:this.Pos])}
	this.Start = this.Pos
}

/*
Returns a token with error information.
*/
func (this *Lexer) Errorf(format string, args ...interface{}) LexFn {
	this.Tokens <- lexertoken.Token{
		Type:  lexertoken.TOKEN_ERROR,
		Value: fmt.Sprintf(format, args...),
	}

	return nil
}

/*
Ignores the current token by setting the lexer's start
position to the current reading position.
*/
func (this *Lexer) Ignore() {
	this.Start = this.Pos
}

/*
Increment the position
*/
func (this *Lexer) Inc() {
	this.Pos++
	if this.Pos >= utf8.RuneCountInString(this.Input) {
		this.Emit(lexertoken.TOKEN_EOF)
	}
}

/*
Return a slice of the input from the current lexer position
to the end of the input string.
*/
func (this *Lexer) InputToEnd() string {
	return this.Input[this.Pos:]
}

/*
Returns the true/false if the lexer is at the end of the
input stream.
*/
func (this *Lexer) IsEOF() bool {
	return this.Pos >= len(this.Input)
}

/*
Returns true/false if then next character is whitespace
*/
func (this *Lexer) IsWhitespace() bool {
	ch, _ := utf8.DecodeRuneInString(this.Input[this.Pos:])
	return unicode.IsSpace(ch)
}

/*
Reads the next rune (character) from the input stream
and advances the lexer position.
*/
func (this *Lexer) Next() rune {
	if this.Pos >= utf8.RuneCountInString(this.Input) {
		this.Width = 0
		return lexertoken.EOF
	}

	result, width := utf8.DecodeRuneInString(this.Input[this.Pos:])

	this.Width = width
	this.Pos += this.Width
	return result
}

/*
Return the next token from the channel
*/

func (this *Lexer) NextToken() chan lexertoken.Token {
	return this.Tokens
}

/*
func (this *Lexer) NextToken() (t lexertoken.Token) {

	defer func() {
		if r := recover(); r != nil {
			t = lexertoken.Token{
				Type: lexertoken.TOKEN_EOF,
			}
		}
	}()

	for {
		select {
		case token := <-this.Tokens:
			return token
		default:
			this.State = this.State(this)
		}
	}
}*/

/*
Returns the next rune in the stream, then puts the lexer
position back. Basically reads the next rune without consuming
it.
*/
func (this *Lexer) Peek() rune {
	rune := this.Next()
	this.Backup()
	return rune
}

/*
Starts the lexical analysis and feeding tokens into the
token channel.
*/
func (this *Lexer) Run() {
	for state := LexBegin; state != nil; {
		state = state(this)
	}

	this.Shutdown()
}

/*
Shuts down the token stream
*/
func (this *Lexer) Shutdown() {
	close(this.Tokens)
}

/*
Skips whitespace until we get something meaningful.
*/
func (this *Lexer) SkipWhitespace() {
	for {
		ch := this.Next()

		if !unicode.IsSpace(ch) {
			this.Dec()
			break
		}

		if ch == lexertoken.EOF {
			this.Emit(lexertoken.TOKEN_EOF)
			break
		}
	}
}

func (this *Lexer) IsCharacter() bool {
	ch := this.Next()
	return unicode.IsLetter(ch) || string(ch) == ":"
}
