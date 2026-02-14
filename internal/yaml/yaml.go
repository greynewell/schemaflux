// Package yaml implements a minimal YAML parser sufficient for pssg config
// files and markdown frontmatter. It supports key-value pairs, nested maps,
// lists of scalars, lists of maps, quoted strings, comments, booleans,
// integers, and floats. It does NOT support anchors, aliases, multi-line
// strings (|, >), flow mappings ({}), or flow sequences ([]).
package yaml

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ---------- public API ----------

// Unmarshal parses YAML data and stores the result in the value pointed to
// by v. It supports struct types with `yaml:"name"` tags and maps.
func Unmarshal(data []byte, v interface{}) error {
	node, err := parse(string(data))
	if err != nil {
		return err
	}
	return decode(node, reflect.ValueOf(v))
}

// UnmarshalMap parses YAML data into a generic map[string]interface{}.
// List values become []interface{}, nested maps become map[string]interface{},
// and scalars are kept as strings (or bool/int/float64 when unambiguous).
func UnmarshalMap(data []byte) (map[string]interface{}, error) {
	node, err := parse(string(data))
	if err != nil {
		return nil, err
	}
	m, ok := nodeToInterface(node).(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("yaml: top-level value is not a mapping")
	}
	return m, nil
}

// ---------- internal AST ----------

// nodeKind distinguishes the three shapes a YAML value can take.
type nodeKind int

const (
	kindScalar nodeKind = iota
	kindMapping
	kindSequence
)

// node is a lightweight AST node.
type node struct {
	kind     nodeKind
	scalar   string          // only for kindScalar
	quoted   bool            // true if the original value was in quotes
	mapping  []mappingEntry  // only for kindMapping (order-preserving)
	sequence []*node         // only for kindSequence
}

type mappingEntry struct {
	key   string
	value *node
}

// ---------- parser ----------

// line is a pre-processed source line.
type line struct {
	indent  int
	content string // leading whitespace stripped
}

// parse turns the raw YAML text into an AST node tree.
func parse(text string) (*node, error) {
	lines := prepareLines(text)
	n, _, err := parseMapping(lines, 0, 0)
	return n, err
}

// prepareLines splits, strips comments, drops blank/comment-only lines, and
// records indentation.
func prepareLines(text string) []line {
	raw := strings.Split(text, "\n")
	out := make([]line, 0, len(raw))
	for _, r := range raw {
		if len(strings.TrimSpace(r)) == 0 {
			continue
		}
		indent := countIndent(r)
		content := stripInlineComment(strings.TrimSpace(r))
		if content == "" || content[0] == '#' {
			continue
		}
		out = append(out, line{indent: indent, content: content})
	}
	return out
}

// countIndent returns the number of leading spaces.
func countIndent(s string) int {
	n := 0
	for _, c := range s {
		if c == ' ' {
			n++
		} else if c == '\t' {
			n += 2 // treat tab as 2 spaces
		} else {
			break
		}
	}
	return n
}

// stripInlineComment removes a trailing # comment from a line, being careful
// not to strip inside quoted strings.
func stripInlineComment(s string) string {
	inSingle := false
	inDouble := false
	for i, c := range s {
		switch c {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble {
				// Only strip if preceded by whitespace (or start).
				if i > 0 && s[i-1] == ' ' {
					return strings.TrimRight(s[:i], " ")
				}
				if i == 0 {
					return ""
				}
			}
		}
	}
	return s
}

// parseMapping parses a block mapping starting at lines[pos] with the given
// expected indent level. It returns the mapping node and the next line index.
func parseMapping(lines []line, pos, indent int) (*node, int, error) {
	n := &node{kind: kindMapping}
	for pos < len(lines) {
		l := lines[pos]
		if l.indent < indent {
			break // dedented past this mapping
		}
		if l.indent > indent {
			break // indented too much — should not happen at top of a mapping
		}

		// List item at this indent level? Then we are actually a sequence.
		if strings.HasPrefix(l.content, "- ") || l.content == "-" {
			// Re-parse as sequence.
			return parseSequence(lines, pos, indent)
		}

		key, val, ok := splitKeyValue(l.content)
		if !ok {
			pos++
			continue
		}

		if val != "" {
			// Inline scalar value.
			s, q := unquote(val)
			n.mapping = append(n.mapping, mappingEntry{key: key, value: &node{kind: kindScalar, scalar: s, quoted: q}})
			pos++
		} else {
			// Value is on subsequent indented lines — could be mapping, sequence, or empty.
			pos++
			if pos < len(lines) && lines[pos].indent > indent {
				childIndent := lines[pos].indent
				if strings.HasPrefix(lines[pos].content, "- ") || lines[pos].content == "-" {
					child, next, err := parseSequence(lines, pos, childIndent)
					if err != nil {
						return nil, next, err
					}
					n.mapping = append(n.mapping, mappingEntry{key: key, value: child})
					pos = next
				} else {
					child, next, err := parseMapping(lines, pos, childIndent)
					if err != nil {
						return nil, next, err
					}
					n.mapping = append(n.mapping, mappingEntry{key: key, value: child})
					pos = next
				}
			} else {
				// Empty value.
				n.mapping = append(n.mapping, mappingEntry{key: key, value: &node{kind: kindScalar, scalar: ""}})
			}
		}
	}
	return n, pos, nil
}

// parseSequence parses a block sequence (lines starting with "- ").
func parseSequence(lines []line, pos, indent int) (*node, int, error) {
	n := &node{kind: kindSequence}
	for pos < len(lines) {
		l := lines[pos]
		if l.indent < indent {
			break
		}
		if l.indent > indent {
			break
		}
		if !strings.HasPrefix(l.content, "- ") && l.content != "-" {
			break
		}

		// Content after "- "
		rest := ""
		if len(l.content) > 2 {
			rest = l.content[2:]
		}

		// Check if the rest is a key: value pair (list of maps).
		if key, val, ok := splitKeyValue(rest); ok {
			// This is a list-of-maps item. Collect all indented lines belonging
			// to this item, plus the first key-value we already have.
			mapNode := &node{kind: kindMapping}
			mapNode.mapping = append(mapNode.mapping, mappingEntry{
				key:   key,
				value: scalarOrEmpty(val),
			})
			pos++
			// Subsequent lines at indent+2 (or more) belong to this map item.
			itemIndent := indent + 2
			for pos < len(lines) && lines[pos].indent >= itemIndent && !strings.HasPrefix(lines[pos].content, "- ") {
				kk, vv, ok2 := splitKeyValue(lines[pos].content)
				if ok2 {
					mapNode.mapping = append(mapNode.mapping, mappingEntry{
						key:   kk,
						value: scalarOrEmpty(vv),
					})
				}
				pos++
			}
			n.sequence = append(n.sequence, mapNode)
		} else {
			// Simple scalar list item.
			s, q := unquote(rest)
			n.sequence = append(n.sequence, &node{kind: kindScalar, scalar: s, quoted: q})
			pos++
		}
	}
	return n, pos, nil
}

// scalarOrEmpty creates a scalar node; if val is empty the scalar is "".
func scalarOrEmpty(val string) *node {
	s, q := unquote(val)
	return &node{kind: kindScalar, scalar: s, quoted: q}
}

// splitKeyValue splits "key: value" into (key, value, true). If the line
// doesn't look like a key-value pair it returns ("", "", false).
func splitKeyValue(s string) (string, string, bool) {
	// Find the first colon that is followed by a space or is at the end.
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			if i+1 == len(s) {
				return s[:i], "", true
			}
			if s[i+1] == ' ' {
				key := s[:i]
				val := strings.TrimSpace(s[i+2:])
				return key, val, true
			}
		}
	}
	return "", "", false
}

// unquote strips matching single or double quotes from a string.
// It returns the unquoted string and whether quotes were present.
func unquote(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1], true
		}
	}
	return s, false
}

// ---------- nodeToInterface ----------

// nodeToInterface converts the AST into plain Go types for UnmarshalMap.
func nodeToInterface(n *node) interface{} {
	switch n.kind {
	case kindScalar:
		if n.quoted {
			return n.scalar // quoted values are always strings
		}
		return parseScalar(n.scalar)
	case kindMapping:
		m := make(map[string]interface{}, len(n.mapping))
		for _, e := range n.mapping {
			m[e.key] = nodeToInterface(e.value)
		}
		return m
	case kindSequence:
		s := make([]interface{}, len(n.sequence))
		for i, child := range n.sequence {
			s[i] = nodeToInterface(child)
		}
		return s
	}
	return nil
}

// parseScalar converts a raw string into a typed Go value: bool, int, float64,
// or string. An empty string stays as "".
func parseScalar(s string) interface{} {
	if s == "" {
		return ""
	}
	switch strings.ToLower(s) {
	case "true":
		return true
	case "false":
		return false
	case "null", "~":
		return nil
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return int(i)
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		// Only return float if it contains a dot to avoid converting "123" to float.
		if strings.Contains(s, ".") {
			return f
		}
	}
	return s
}

// ---------- struct decoder ----------

// decode writes the AST into a reflect.Value which must be a pointer.
func decode(n *node, rv reflect.Value) error {
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("yaml: Unmarshal requires a non-nil pointer")
	}
	return decodeValue(n, rv.Elem())
}

// decodeValue recursively fills rv from the AST node.
func decodeValue(n *node, rv reflect.Value) error {
	// If rv is a pointer, allocate and recurse.
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		return decodeValue(n, rv.Elem())
	}

	// Interface — store the generic Go form.
	if rv.Kind() == reflect.Interface {
		rv.Set(reflect.ValueOf(nodeToInterface(n)))
		return nil
	}

	switch n.kind {
	case kindScalar:
		return setScalar(rv, n.scalar)

	case kindMapping:
		if rv.Kind() == reflect.Struct {
			return decodeStruct(n, rv)
		}
		if rv.Kind() == reflect.Map {
			return decodeMap(n, rv)
		}
		return fmt.Errorf("yaml: cannot decode mapping into %s", rv.Type())

	case kindSequence:
		return decodeSlice(n, rv)
	}

	return nil
}

// decodeStruct fills a struct value using yaml struct tags.
func decodeStruct(n *node, rv reflect.Value) error {
	rt := rv.Type()
	tagMap := buildTagMap(rt)

	for _, e := range n.mapping {
		idx, ok := tagMap[e.key]
		if !ok {
			continue // ignore unknown keys
		}
		field := rv.Field(idx)
		if !field.CanSet() {
			continue
		}
		if err := decodeValue(e.value, field); err != nil {
			return fmt.Errorf("yaml: field %q: %w", e.key, err)
		}
	}
	return nil
}

// buildTagMap returns a map from yaml tag name to struct field index.
func buildTagMap(rt reflect.Type) map[string]int {
	m := make(map[string]int, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		tag := f.Tag.Get("yaml")
		if tag == "" || tag == "-" {
			continue
		}
		// Strip options like ",omitempty".
		if idx := strings.Index(tag, ","); idx >= 0 {
			tag = tag[:idx]
		}
		m[tag] = i
	}
	return m
}

// decodeMap fills a map[string]T value.
func decodeMap(n *node, rv reflect.Value) error {
	if rv.IsNil() {
		rv.Set(reflect.MakeMap(rv.Type()))
	}
	valType := rv.Type().Elem()
	for _, e := range n.mapping {
		val := reflect.New(valType).Elem()
		if err := decodeValue(e.value, val); err != nil {
			return err
		}
		rv.SetMapIndex(reflect.ValueOf(e.key), val)
	}
	return nil
}

// decodeSlice fills a slice value.
func decodeSlice(n *node, rv reflect.Value) error {
	elemType := rv.Type().Elem()
	slice := reflect.MakeSlice(rv.Type(), 0, len(n.sequence))
	for _, child := range n.sequence {
		elem := reflect.New(elemType).Elem()
		if err := decodeValue(child, elem); err != nil {
			return err
		}
		slice = reflect.Append(slice, elem)
	}
	rv.Set(slice)
	return nil
}

// setScalar sets a scalar reflect value from a string.
func setScalar(rv reflect.Value, s string) error {
	s = strings.TrimSpace(s)
	switch rv.Kind() {
	case reflect.String:
		rv.SetString(s)
	case reflect.Bool:
		rv.SetBool(strings.EqualFold(s, "true"))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if s == "" {
			rv.SetInt(0)
			return nil
		}
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("yaml: cannot parse %q as int: %w", s, err)
		}
		rv.SetInt(i)
	case reflect.Float32, reflect.Float64:
		if s == "" {
			rv.SetFloat(0)
			return nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("yaml: cannot parse %q as float: %w", s, err)
		}
		rv.SetFloat(f)
	case reflect.Interface:
		rv.Set(reflect.ValueOf(parseScalar(s)))
	default:
		return fmt.Errorf("yaml: cannot set scalar into %s", rv.Type())
	}
	return nil
}
