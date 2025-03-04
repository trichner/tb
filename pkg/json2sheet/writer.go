package json2sheet

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/trichner/tb/pkg/jsontree"
	"github.com/trichner/tb/pkg/jsontree/ast"
	"github.com/trichner/tb/pkg/jsontree/lexer"
)

func WriteArraysTo(to SheetUpdater, from io.Reader) error {
	rows, err := mapArraysToRows(from)
	if err != nil {
		return err
	}

	return to.UpdateValues(rows)
}

func WriteObjectsTo(to SheetUpdater, from io.Reader) error {
	rows, err := mapObjectsToRows(from)
	if err != nil {
		return err
	}

	return to.UpdateValues(rows)
}

func AppendArraysTo(to SheetAppender, from io.Reader) error {
	rows, err := mapArraysToRows(from)
	if err != nil {
		return err
	}

	return to.AppendValues(rows)
}

func AppendObjectsTo(to SheetAppender, from io.Reader) error {
	rows, err := mapObjectsToRows(from)
	if err != nil {
		return err
	}

	return to.AppendValues(rows)
}

func mapObjectsToRows(from io.Reader) ([][]string, error) {
	var rows [][]string

	l := lexer.NewLexer(from)

	// write empty header row for a start
	rows = append(rows, []string{})

	headers := map[string]int{}

	for {
		root, err := jsontree.Parse(l)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if root.Type() != ast.NodeTypeObject {
			return nil, fmt.Errorf("json is not an object: %s", root.Type())
		}

		node := root.(ast.ObjectNode)

		headers = appendToHeaderMap(node, headers)

		row := toRow(node, headers)
		rows = append(rows, row)
	}

	rows[0] = headersToRow(headers)
	return rows, nil
}

func mapArraysToRows(from io.Reader) ([][]string, error) {
	var rows [][]string
	l := lexer.NewLexer(from)
	for {
		root, err := jsontree.Parse(l)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if root.Type() != ast.NodeTypeArray {
			return nil, fmt.Errorf("json object is not an array: %s", root.Type())
		}

		node := root.(ast.ArrayNode)
		row := make([]string, len(node.Items()))
		for i, v := range node.Items() {
			row[i] = toString(v)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func headersToRow(headers map[string]int) []string {
	row := make([]string, len(headers))
	for k, v := range headers {
		row[v] = k
	}
	return row
}

func appendToHeaderMap(root ast.ObjectNode, headers map[string]int) map[string]int {
	for _, v := range root.Properties() {
		_, ok := headers[v.Name]
		if !ok {
			headers[v.Name] = len(headers)
		}
	}
	return headers
}

func toRow(root ast.ObjectNode, headers map[string]int) []string {
	row := make([]string, len(headers))
	for _, v := range root.Properties() {
		idx, ok := headers[v.Name]
		if !ok {
			panic(fmt.Errorf("unexpected header name: %s", v.Name))
		}
		row[idx] = toString(v.Value)
	}
	return row
}

func toString(n ast.Node) string {
	switch n.Type() {
	case ast.NodeTypeBoolean:
		typed := n.(ast.BooleanNode)
		if typed.Value() {
			return "TRUE"
		} else {
			return "FALSE"
		}
	case ast.NodeTypeNull:
		return ""
	case ast.NodeTypeNumber:
		typed := n.(ast.NumberNode)
		return typed.Value()
	case ast.NodeTypeText:
		typed := n.(ast.TextNode)
		return typed.Value()
	default:
		bytes, err := json.Marshal(n)
		if err == nil {
			return string(bytes)
		}
		return fmt.Sprintf("%s", n)
	}
}
