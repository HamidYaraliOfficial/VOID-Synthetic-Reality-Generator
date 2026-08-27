// Package config loads Versioned scenario/universe configuration from JSON
// or a lightweight YAML subset (block mappings, block sequences, scalars —
// enough for VOID's own config files without pulling in an external YAML
// dependency, keeping the whole backend buildable with zero network access).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Root is the top-level shape of a VOID config file (scenario, universe
// seed, network profile, etc. all nest under here as free-form maps so the
// schema can evolve without breaking old files).
type Root struct {
	Version  int                    `json:"version" yaml:"version"`
	Kind     string                 `json:"kind" yaml:"kind"` // scenario | universe | dashboard
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec     map[string]interface{} `json:"spec" yaml:"spec"`
}

// Load reads a config file, choosing JSON or the built-in YAML-subset parser
// based on file extension (.json vs .yaml/.yml).
func Load(path string) (*Root, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		var r Root
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("config: parsing JSON %s: %w", path, err)
		}
		return &r, nil
	case ".yaml", ".yml":
		m, err := ParseYAML(string(data))
		if err != nil {
			return nil, fmt.Errorf("config: parsing YAML %s: %w", path, err)
		}
		return fromMap(m), nil
	default:
		return nil, fmt.Errorf("config: unsupported extension %q (use .json/.yaml/.yml)", ext)
	}
}

func fromMap(m map[string]interface{}) *Root {	r := &Root{Spec: map[string]interface{}{}}
	if v, ok := m["version"]; ok {
		if f, ok := v.(float64); ok {
			r.Version = int(f)
		}
	}
	if v, ok := m["kind"].(string); ok {
		r.Kind = v
	}
	if v, ok := m["metadata"].(map[string]interface{}); ok {
		r.Metadata = v
	}
	if v, ok := m["spec"].(map[string]interface{}); ok {
		r.Spec = v
	}
	return r
}

// --- minimal YAML-subset parser --------------------------------------------
//
// Supports: block mappings ("key: value"), nested mappings via 2-space
// indentation, block sequences ("- item" / "- key: value"), scalars
// (string/int/float/bool/null), and '#' comments. This intentionally does
// NOT implement flow style, anchors, or multi-document streams — VOID's own
// config files stick to this subset, and it keeps the backend dependency-free.

// ParseYAML parses a restricted YAML document into a generic map.
func ParseYAML(src string) (map[string]interface{}, error) {
	lines := stripCommentsAndBlank(src)
	val, _, err := parseBlock(lines, 0, 0)
	if err != nil {
		return nil, err
	}
	m, ok := val.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"value": val}, nil
	}
	return m, nil
}

type yline struct {
	indent int
	text   string
}

func stripCommentsAndBlank(src string) []yline {
	var out []yline
	for _, raw := range strings.Split(src, "\n") {
		line := raw
		if idx := strings.Index(line, "#"); idx >= 0 {
			// naive comment strip (fine for our own config subset; quoted
			// strings containing '#' are not expected in VOID configs).
			line = line[:idx]
		}
		trimmedRight := strings.TrimRight(line, " \t\r")
		if strings.TrimSpace(trimmedRight) == "" {
			continue
		}
		indent := 0
		for indent < len(trimmedRight) && trimmedRight[indent] == ' ' {
			indent++
		}
		out = append(out, yline{indent: indent, text: strings.TrimSpace(trimmedRight)})
	}
	return out
}

// parseBlock parses lines[start:] at the given indent level, returning the
// parsed value and the index of the first unconsumed line.
func parseBlock(lines []yline, start, indent int) (interface{}, int, error) {
	if start >= len(lines) {
		return nil, start, nil
	}
	if strings.HasPrefix(lines[start].text, "- ") || lines[start].text == "-" {
		return parseSequence(lines, start, indent)
	}
	return parseMapping(lines, start, indent)
}

func parseMapping(lines []yline, start, indent int) (interface{}, int, error) {
	m := map[string]interface{}{}
	i := start
	for i < len(lines) {
		ln := lines[i]
		if ln.indent < indent {
			break
		}
		if ln.indent > indent {
			return nil, i, fmt.Errorf("yaml: unexpected indent at line %q", ln.text)
		}
		colon := findTopLevelColon(ln.text)
		if colon < 0 {
			return nil, i, fmt.Errorf("yaml: expected 'key: value' at %q", ln.text)
		}
		key := strings.TrimSpace(ln.text[:colon])
		rest := strings.TrimSpace(ln.text[colon+1:])
		key = trimQuotes(key)

		if rest == "" {
			// nested block on following, more-indented lines
			if i+1 < len(lines) && lines[i+1].indent > indent {
				child, next, err := parseBlock(lines, i+1, lines[i+1].indent)
				if err != nil {
					return nil, i, err
				}
				m[key] = child
				i = next
				continue
			}
			m[key] = nil
			i++
			continue
		}
		m[key] = parseScalar(rest)
		i++
	}
	return m, i, nil
}

func parseSequence(lines []yline, start, indent int) (interface{}, int, error) {
	var seq []interface{}
	i := start
	for i < len(lines) {
		ln := lines[i]
		if ln.indent != indent || !strings.HasPrefix(ln.text, "-") {
			break
		}
		item := strings.TrimSpace(strings.TrimPrefix(ln.text, "-"))
		if item == "" {
			if i+1 < len(lines) && lines[i+1].indent > indent {
				child, next, err := parseBlock(lines, i+1, lines[i+1].indent)
				if err != nil {
					return nil, i, err
				}
				seq = append(seq, child)
				i = next
				continue
			}
			seq = append(seq, nil)
			i++
			continue
		}
		if colon := findTopLevelColon(item); colon >= 0 {
			// inline "- key: value" starts a mapping item; feed it back
			// through parseMapping by synthesizing an indented sub-block.
			synthetic := []yline{{indent: indent + 2, text: item}}
			j := i + 1
			for j < len(lines) && lines[j].indent > indent {
				synthetic = append(synthetic, lines[j])
				j++
			}
			child, _, err := parseMapping(synthetic, 0, indent+2)
			if err != nil {
				return nil, i, err
			}
			seq = append(seq, child)
			i = j
			continue
		}
		seq = append(seq, parseScalar(item))
		i++
	}
	return seq, i, nil
}

func findTopLevelColon(s string) int {
	inQuotes := false
	for i, r := range s {
		if r == '"' {
			inQuotes = !inQuotes
		}
		if r == ':' && !inQuotes {
			if i+1 == len(s) || s[i+1] == ' ' {
				return i
			}
		}
	}
	return -1
}

func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func parseScalar(s string) interface{} {
	s = trimQuotes(strings.TrimSpace(s))
	switch s {
	case "true":
		return true
	case "false":
		return false
	case "null", "~", "":
		return nil
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return float64(i)
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

// DecodeSpec re-marshals Root.Spec to JSON and decodes it into v, letting
// CLI/API callers turn a generic parsed config file into a concrete typed
// struct (entity.Schema, scenario.Scenario, ...) without duplicating parsing
// logic per format.
func DecodeSpec(spec map[string]interface{}, v interface{}) error {
	data, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
