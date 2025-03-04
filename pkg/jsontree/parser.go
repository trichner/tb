package jsontree

import (
	"fmt"
	"io"

	"github.com/trichner/tb/pkg/jsontree/ast"
	"github.com/trichner/tb/pkg/jsontree/lexer"
)

func Parse(l lexer.Lexer) (ast.Node, error) {
	token, err := l.Peek()
	if err != nil {
		return nil, err
	}

	switch token.Type {
	case lexer.TokenTypeEOF:
		return nil, io.EOF
	case lexer.TokenTypeOpeningBrace:
		return parseObject(l)
	case lexer.TokenTypeOpeningBracket:
		return parseArray(l)
	case lexer.TokenTypePrimitiveNumber:
		return parseNumber(l)
	case lexer.TokenTypePrimitiveText:
		return parsePrimitiveText(l)
	case lexer.TokenTypeText:
		return parseText(l)
	}
	return nil, fmt.Errorf("unexpected token: %v", token)
}

func parseArray(l lexer.Lexer) (ast.Node, error) {
	err := skipToken(l, lexer.TokenTypeOpeningBracket)
	if err != nil {
		return nil, err
	}

	// check for empty array
	peeked, err := l.Peek()
	if err != nil {
		return nil, fmt.Errorf("unexpected error parsing array: %w", err)
	}
	if peeked.Type == lexer.TokenTypeClosingBracket {
		// discard closing bracket
		_, err = l.Token()
		return ast.NewArrayNode(nil), err
	}

	var items []ast.Node
	for {
		item, err := Parse(l)
		if err != nil {
			return nil, fmt.Errorf("unexpected error parsing array item: %w", err)
		}
		items = append(items, item)

		sep, err := l.Token()
		if err != nil {
			return nil, fmt.Errorf("unexpected error parsing array: %w", err)
		}
		if sep.Type == lexer.TokenTypeClosingBracket {
			break
		}
		if sep.Type != lexer.TokenTypeComma {
			return nil, fmt.Errorf("unexpected error parsing array, expected comma but got: %v", sep.Type)
		}
	}

	return ast.NewArrayNode(items), err
}

func parseText(l lexer.Lexer) (ast.TextNode, error) {
	token, err := l.Token()
	if err != nil {
		return nil, err
	}

	return ast.NewTextNode(token.Value), nil
}

func parsePrimitiveText(l lexer.Lexer) (ast.Node, error) {
	token, err := l.Token()
	if err != nil {
		return nil, err
	}

	switch token.Value {
	case "true":
		return ast.NewBooleanNode(true), nil
	case "false":
		return ast.NewBooleanNode(false), nil
	case "null":
		return ast.NewNullNode(), nil
	}
	return nil, fmt.Errorf("unrecognized literal: %q", token.Value)
}

func parseNumber(l lexer.Lexer) (ast.Node, error) {
	tkn, err := l.Token()
	if err != nil {
		return nil, fmt.Errorf("unexpected token parsing number: %w", err)
	}
	if tkn.Type != lexer.TokenTypePrimitiveNumber {
		return nil, fmt.Errorf("unexpected token, expected %s: %s", lexer.TokenTypePrimitiveNumber, tkn.Type)
	}
	return ast.NewNumberNode(tkn.Value), nil
}

func parseObject(l lexer.Lexer) (ast.Node, error) {
	err := skipToken(l, lexer.TokenTypeOpeningBrace)
	if err != nil {
		return nil, wrapUnexpectedObjectParseException(err)
	}

	tkn, err := l.Peek()
	if err != nil {
		return nil, wrapUnexpectedObjectParseException(err)
	}
	if tkn.Type == lexer.TokenTypeClosingBrace {
		err := skipToken(l, lexer.TokenTypeClosingBrace)
		return ast.NewObjectNode(nil), err
	}

	var properties []*ast.Property

	for {
		property, err := parseObjectPropertyName(l)
		if err != nil {
			return nil, err
		}

		err = skipToken(l, lexer.TokenTypeColon)
		if err != nil {
			return nil, wrapUnexpectedObjectParseException(err)
		}

		val, err := Parse(l)
		if err != nil {
			return nil, fmt.Errorf("unexpected error parsing object value: %w", err)
		}

		properties = append(properties, &ast.Property{
			Name:  property,
			Value: val,
		})

		tkn, err = l.Token()
		if err != nil {
			return nil, wrapUnexpectedObjectParseException(err)
		}
		if tkn.Type == lexer.TokenTypeClosingBrace {
			break
		}
		if tkn.Type != lexer.TokenTypeComma {
			return nil, fmt.Errorf("unexpected token parsing object, expected %q but got: %v", lexer.TokenTypeComma, tkn)
		}
	}

	return ast.NewObjectNode(properties), nil
}

func parseObjectPropertyName(l lexer.Lexer) (string, error) {
	tkn, err := l.Token()
	if err != nil {
		return "", wrapUnexpectedObjectParseException(err)
	}
	if tkn.Type != lexer.TokenTypeText {
		return "", fmt.Errorf("unexpected token parsing object, expected %q but got: %q", lexer.TokenTypeText, tkn.Type)
	}
	return tkn.Value, nil
}

func skipToken(l lexer.Lexer, t lexer.TokenType) error {
	tkn, err := l.Token()
	if err != nil {
		return fmt.Errorf("expected token %q but got: %w", t, err)
	}
	if tkn.Type != t {
		return fmt.Errorf("unexpected token, expected %q but got: %v", t, tkn)
	}
	return nil
}

func wrapUnexpectedObjectParseException(err error) error {
	return fmt.Errorf("unexpected error parsing object: %w", err)
}
