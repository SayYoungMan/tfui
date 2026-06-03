package jsondiff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type Options struct {
	Indent       string              // indent used for result json string (default: "  ")
	AddedStyle   func(string) string // style function used to style line added (default: str -> + str)
	RemovedStyle func(string) string // style function used to style line removed (default: str -> - str)
	NormalStyle  func(string) string // style function used to style normal lines (defualt: str ->   str)
}

type LineKind int

const (
	LineNormal LineKind = iota
	LineAdded
	LineRemoved
)

type Line struct {
	Kind LineKind
	Text string
}

type Result struct {
	Lines []Line
	opts  Options
}

type diffKind int

const (
	diffSame diffKind = iota
	diffAdded
	diffRemoved
	diffChanged
)

type diffNode struct {
	kind   diffKind
	before any
	after  any
	object map[string]*diffNode
	array  []*diffNode
}

// takes two JSON raw messages and compare them and then render their diffs as a string
func Render(before, after json.RawMessage, opts Options) (string, error) {
	res, err := RenderResult(before, after, opts)
	if err != nil {
		return "", err
	}
	return res.String(), nil
}

// Provide sensible default to options if not specified
func (o *Options) normalize() {
	if o.Indent == "" {
		o.Indent = "  "
	}

	if o.AddedStyle == nil {
		o.AddedStyle = func(s string) string {
			return "+ " + s
		}
	}

	if o.RemovedStyle == nil {
		o.RemovedStyle = func(s string) string {
			return "- " + s
		}
	}

	if o.NormalStyle == nil {
		o.NormalStyle = func(s string) string {
			return "  " + s
		}
	}
}

func RenderResult(before, after json.RawMessage, opts Options) (Result, error) {
	opts.normalize()

	beforeVal, err := decodeJSON(before)
	if err != nil {
		return Result{}, fmt.Errorf("failed to parse before JSON: %w", err)
	}
	afterVal, err := decodeJSON(after)
	if err != nil {
		return Result{}, fmt.Errorf("failed to parse after JSON: %w", err)
	}

	root := buildRoot(beforeVal, afterVal)
	res := &Result{opts: opts}

	err = res.renderNode(root, nil, 0, false)
	if err != nil {
		return Result{}, err
	}

	return *res, nil
}

func decodeJSON(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber() // This is to preserve number as it is instead of converting to float64

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}

	return value, nil
}

func buildRoot(before, after any) *diffNode {
	if reflect.DeepEqual(before, after) {
		return &diffNode{kind: diffSame, after: after}
	}

	// For the cases where whole resource is created/removed we don't want to show - null
	// Therefore, explicitly drop the before/after if it's nil for the root node
	if before != nil && after == nil {
		return &diffNode{kind: diffRemoved, before: before}
	}
	if before == nil && after != nil {
		return &diffNode{kind: diffAdded, after: after}
	}

	return buildDiff(before, after)
}

// construct tree of JSON diff object nodes recursively and return the root
func buildDiff(before, after any) *diffNode {
	if reflect.DeepEqual(before, after) {
		return &diffNode{kind: diffSame, after: after}
	}

	// If they are both object, diff each one of the keys in sorted order
	beforeObject, isBeforeObject := before.(map[string]any)
	afterObject, isAfterObject := after.(map[string]any)
	if isBeforeObject && isAfterObject {
		node := &diffNode{kind: diffChanged, object: make(map[string]*diffNode)}
		for _, key := range sortedUnionKeys(beforeObject, afterObject) {
			beforeValue, beforeOK := beforeObject[key]
			afterValue, afterOK := afterObject[key]

			switch {
			case !beforeOK:
				node.object[key] = &diffNode{kind: diffAdded, after: afterValue}
			case !afterOK:
				node.object[key] = &diffNode{kind: diffRemoved, before: beforeValue}
			default: // This is essentially the case beforeOK && afterOK
				node.object[key] = buildDiff(beforeValue, afterValue)
			}
		}
		return node
	}

	// Recurse and diff arrays only if they have the same length for now
	// TODO: Use better algorithm to support diffing with different lengths
	beforeArray, beforeIsArray := before.([]any)
	afterArray, afterIsArray := after.([]any)
	if beforeIsArray && afterIsArray && len(beforeArray) == len(afterArray) {
		node := &diffNode{kind: diffChanged}
		for i := range beforeArray {
			node.array = append(node.array, buildDiff(beforeArray[i], afterArray[i]))
		}
		return node
	}

	return &diffNode{kind: diffChanged, before: before, after: after}
}

// renders Result into a syntax highlighted JSON string
// errors during syntax highlight will be consumed and return string with no highlight for that line
func (r Result) String() string {
	var b strings.Builder
	for _, line := range r.Lines {
		text := line.Text

		switch line.Kind {
		case LineAdded:
			text = r.opts.AddedStyle(text)
		case LineRemoved:
			text = r.opts.RemovedStyle(text)
		default:
			text = r.opts.NormalStyle(text)
		}

		b.WriteString(text)
		b.WriteByte('\n')
	}

	return b.String()
}

func (r *Result) renderNode(node *diffNode, key *string, indentLevel int, comma bool) error {
	switch node.kind {
	case diffSame:
		return r.renderPlain(LineNormal, key, node.after, indentLevel, comma)
	case diffAdded:
		return r.renderPlain(LineAdded, key, node.after, indentLevel, comma)
	case diffRemoved:
		return r.renderPlain(LineRemoved, key, node.before, indentLevel, comma)
	case diffChanged:
		switch {
		case node.object != nil:
			return r.renderObject(node, key, indentLevel, comma)
		case node.array != nil:
			return r.renderArray(node, key, indentLevel, comma)
		default:
			if err := r.renderPlain(LineRemoved, key, node.before, indentLevel, comma); err != nil {
				return err
			}
			return r.renderPlain(LineAdded, key, node.after, indentLevel, comma)
		}
	}
	return nil
}

func (r *Result) renderPlain(kind LineKind, key *string, value any, indentLevel int, comma bool) error {
	switch v := value.(type) {
	case map[string]any:
		r.appendLine(kind, indentLevel, formatKey(key)+"{")
		keys := sortedKeys(v)
		for i, k := range keys {
			keyCopy := k
			if err := r.renderPlain(kind, &keyCopy, v[k], indentLevel+1, i < len(keys)-1); err != nil {
				return err
			}
		}
		r.appendLine(kind, indentLevel, "}"+commaString(comma))
	case []any:
		r.appendLine(kind, indentLevel, formatKey(key)+"[")
		for i, e := range v {
			if err := r.renderPlain(kind, nil, e, indentLevel+1, i < len(v)-1); err != nil {
				return err
			}
		}
		r.appendLine(kind, indentLevel, "]"+commaString(comma))
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return err
		}
		r.appendLine(kind, indentLevel, formatKey(key)+string(encoded)+commaString(comma))
	}

	return nil
}

// method used to render changed object when some of its nodes changed
func (r *Result) renderObject(node *diffNode, key *string, indentLevel int, comma bool) error {
	r.appendLine(LineNormal, indentLevel, formatKey(key)+"{")

	keys := sortedNodeKeys(node.object)
	for i, k := range keys {
		keyCopy := k
		if err := r.renderNode(node.object[k], &keyCopy, indentLevel+1, i < len(keys)-1); err != nil {
			return err
		}
	}

	r.appendLine(LineNormal, indentLevel, "}"+commaString(comma))
	return nil
}

// method used to render changed array when some of its elements changed
func (r *Result) renderArray(node *diffNode, key *string, indentLevel int, comma bool) error {
	r.appendLine(LineNormal, indentLevel, formatKey(key)+"[")

	for i, e := range node.array {
		if err := r.renderNode(e, nil, indentLevel+1, i < len(node.array)-1); err != nil {
			return err
		}
	}

	r.appendLine(LineNormal, indentLevel, "]"+commaString(comma))
	return nil
}

func (r *Result) appendLine(kind LineKind, indentLevel int, text string) {
	r.Lines = append(r.Lines, Line{
		Kind: kind,
		Text: strings.Repeat(r.opts.Indent, indentLevel) + text,
	})
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// return sorted slice of union of two map keys
func sortedUnionKeys(before, after map[string]any) []string {
	seen := make(map[string]bool, len(before)+len(after))
	for key := range before {
		seen[key] = true
	}
	for key := range after {
		seen[key] = true
	}

	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// return sorted slice of keys of a diffNode
func sortedNodeKeys(values map[string]*diffNode) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// helper to correctly render optional key to JSON format.
// json.Marshal is used to correctly escape some characters and add quotes around
func formatKey(key *string) string {
	if key == nil {
		return ""
	}
	encoded, _ := json.Marshal(key)
	return string(encoded) + ": "
}

func commaString(comma bool) string {
	if comma {
		return ","
	}
	return ""
}
