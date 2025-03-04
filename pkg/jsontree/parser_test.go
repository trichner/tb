package jsontree

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trichner/tb/pkg/jsontree/lexer"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name             string
		raw              string
		expectedErrorMsg string
	}{
		{
			name: "array 1",
			raw:  "[\"hi\"]",
		},
		{
			name: "array 2 empty",
			raw:  "[]",
		},
		{
			name: "array 3 multiple elements",
			raw:  "[\"hi\",\"there\"]",
		},
		{
			name: "array 4 multiple types",
			raw:  "[\"hi\",true]",
		},
		{
			name: "array 5 nested",
			raw:  "[\"hi\",[true,false]]",
		},
		{
			name: "object 1",
			raw:  "{}",
		},
		{
			name: "object 2",
			raw:  "{ \"test\": true }",
		},
		{
			name: "object 3 nested",
			raw:  "{ \"test\": {\"hello\":\"wor{}ld\"} }",
		},
		{
			name: "object 4 multiple",
			raw:  "{ \"test\": true, \"another\": false, \"third\": null }",
		},
		{
			name: "number 1",
			raw:  "1",
		},
		{
			name: "number 2",
			raw:  "[-3,-2,-1,0,1,2,3,1e3,1e4,1e-5]",
		},
		{
			name:             "bad 1",
			raw:              "{",
			expectedErrorMsg: "unexpected token parsing object, expected \"TokenTypeText\" but got: \"TokenTypeEOF\"",
		},
		{
			name:             "bad 2",
			raw:              "[",
			expectedErrorMsg: "unexpected error parsing array item: EOF",
		},
		{
			name:             "bad 3",
			raw:              "hello",
			expectedErrorMsg: "unrecognized literal: \"hello\"",
		},
		{
			name:             "bad 4",
			raw:              "True",
			expectedErrorMsg: "cannot lex token: unrecognized token: 'T'",
		},
		{
			name:             "bad 5",
			raw:              "[[],[]",
			expectedErrorMsg: "unexpected error parsing array, expected comma but got: TokenTypeEOF",
		},
		{
			name:             "bad 6",
			raw:              "{\"hi\":{},,",
			expectedErrorMsg: "unexpected token parsing object, expected \"TokenTypeText\" but got: \"TokenTypeComma\"",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			l := lexer.NewLexer(strings.NewReader(test.raw))
			n, err := Parse(l)
			if test.expectedErrorMsg != "" {
				assert.Error(t, err)
				assert.Equal(t, test.expectedErrorMsg, err.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, n)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	expected := `{"a":"hello","b":null,"c":[2,4,8]}`

	l := lexer.NewLexer(strings.NewReader(expected))
	n, err := Parse(l)
	assert.NoError(t, err)

	actual, _ := n.MarshalJSON()

	assert.Equal(t, expected, string(actual))
}

func TestStream(t *testing.T) {
	expected := `{"a":"hello"}{"b":"world"}`

	l := lexer.NewLexer(strings.NewReader(expected))
	n, err := Parse(l)
	assert.NoError(t, err)

	actual, _ := n.MarshalJSON()

	fmt.Println(string(actual))

	n, err = Parse(l)
	actual, _ = n.MarshalJSON()
	fmt.Println(string(actual))
}
