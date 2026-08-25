package boxl

import (
	"strings"
	"testing"
)

func TestTokenize(t *testing.T) {
	tokens, err := Tokenize("node-one[\"hello\\nworld\"] -> node.two\r\n")
	if err != nil {
		t.Fatal(err)
	}
	wantTypes := []TokenType{
		TokenIdentifier, TokenLeftBracket, TokenString, TokenRightBracket,
		TokenArrow, TokenIdentifier, TokenNewline, TokenEOF,
	}
	if len(tokens) != len(wantTypes) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(wantTypes))
	}
	for i, want := range wantTypes {
		if tokens[i].Type != want {
			t.Errorf("token %d type = %s, want %s", i, tokens[i].Type, want)
		}
	}
	if tokens[0].Value != "node-one" {
		t.Errorf("identifier = %q", tokens[0].Value)
	}
	if tokens[2].Value != "hello\nworld" {
		t.Errorf("quoted value = %q", tokens[2].Value)
	}
	if got := tokens[5].Position; got.Line != 1 || got.Column != 29 {
		t.Errorf("second identifier position = %+v", got)
	}
	if got := tokens[7].Position; got.Line != 2 || got.Column != 1 {
		t.Errorf("EOF position = %+v", got)
	}
}

func TestTokenizerReportsSourcePosition(t *testing.T) {
	_, err := Tokenize("a -> b\n  $")
	if err == nil || !strings.Contains(err.Error(), "line 2, column 3") {
		t.Fatalf("error = %v", err)
	}
}

func TestTokenizerRejectsUnterminatedText(t *testing.T) {
	_, err := Tokenize("a[\"missing]")
	if err == nil || !strings.Contains(err.Error(), "unterminated quoted text") {
		t.Fatalf("error = %v", err)
	}
}

func TestTokenizeSingleQuotedText(t *testing.T) {
	tokens, err := Tokenize("a['hello\\nworld\\'s']")
	if err != nil {
		t.Fatal(err)
	}
	if tokens[2].Type != TokenString || tokens[2].Value != "hello\nworld's" {
		t.Fatalf("quoted token = %+v", tokens[2])
	}
}

func TestTokenizeMultilineQuotedText(t *testing.T) {
	tokens, err := Tokenize("a['first line\r\n    second line'] -> b")
	if err != nil {
		t.Fatal(err)
	}
	if tokens[2].Value != "first line\n    second line" {
		t.Fatalf("quoted value = %q", tokens[2].Value)
	}
	if got := tokens[4].Position; got.Line != 2 || got.Column != 19 {
		t.Fatalf("arrow position = %+v", got)
	}
}

func TestTokenizeLayoutConstraints(t *testing.T) {
	tokens, err := Tokenize("a@then b,below c")
	if err != nil {
		t.Fatal(err)
	}
	want := []TokenType{
		TokenIdentifier, TokenAt, TokenIdentifier, TokenIdentifier, TokenComma,
		TokenIdentifier, TokenIdentifier, TokenEOF,
	}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(want))
	}
	for i, tokenType := range want {
		if tokens[i].Type != tokenType {
			t.Errorf("token %d = %s, want %s", i, tokens[i].Type, tokenType)
		}
	}
}
