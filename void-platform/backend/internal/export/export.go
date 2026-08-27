// Package export implements the Synthetic Dataset Exporter: turning a slice
// of records (already-flattened entity attribute maps) into JSON, JSONL,
// CSV, YAML, XML or a simple length-prefixed custom binary format, with a
// Chunked variant for very large datasets that must not be held fully in
// memory as a single encoded buffer.
package export

import (
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Format enumerates supported export formats.
type Format string

const (
	FormatJSON   Format = "json"
	FormatJSONL  Format = "jsonl"
	FormatCSV    Format = "csv"
	FormatYAML   Format = "yaml"
	FormatXML    Format = "xml"
	FormatSQL    Format = "sql"
	FormatBinary Format = "bin"
)

// Record is one flattened row/document to export.
type Record map[string]interface{}

// ToFile writes records to path in the given format.
func ToFile(path string, format Format, table string, records []Record) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return ToWriter(f, format, table, records)
}

// ToWriter streams records to w in the given format without necessarily
// buffering the entire encoded output, supporting the "Chunked Export for
// very large datasets" requirement.
func ToWriter(w io.Writer, format Format, table string, records []Record) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(records)
	case FormatJSONL:
		enc := json.NewEncoder(w)
		for _, r := range records {
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		return nil
	case FormatCSV:
		return writeCSV(w, records)
	case FormatYAML:
		return writeYAML(w, records)
	case FormatXML:
		return writeXML(w, table, records)
	case FormatSQL:
		return writeSQL(w, table, records)
	case FormatBinary:
		return writeBinary(w, records)
	default:
		return fmt.Errorf("export: unsupported format %q", format)
	}
}

func columnOrder(records []Record) []string {
	set := map[string]bool{}
	for _, r := range records {
		for k := range r {
			set[k] = true
		}
	}
	cols := make([]string, 0, len(set))
	for k := range set {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	return cols
}

func writeCSV(w io.Writer, records []Record) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	cols := columnOrder(records)
	if err := cw.Write(cols); err != nil {
		return err
	}
	for _, r := range records {
		row := make([]string, len(cols))
		for i, c := range cols {
			row[i] = fmt.Sprintf("%v", r[c])
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// writeYAML implements a minimal, dependency-free YAML subset sufficient for
// exporting flat/nested record maps (block-style sequences and mappings).
// It intentionally does not implement the entire YAML 1.2 spec.
func writeYAML(w io.Writer, records []Record) error {
	for _, r := range records {
		if _, err := io.WriteString(w, "-\n"); err != nil {
			return err
		}
		cols := make([]string, 0, len(r))
		for k := range r {
			cols = append(cols, k)
		}
		sort.Strings(cols)
		for _, k := range cols {
			line := fmt.Sprintf("  %s: %s\n", k, yamlScalar(r[k]))
			if _, err := io.WriteString(w, line); err != nil {
				return err
			}
		}
	}
	return nil
}

func yamlScalar(v interface{}) string {
	switch val := v.(type) {
	case string:
		if strings.ContainsAny(val, ":#\n") || val == "" {
			return fmt.Sprintf("%q", val)
		}
		return val
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", val)
	}
}

type xmlRecord struct {
	XMLName xml.Name    `xml:"record"`
	Fields  []xmlField  `xml:"field"`
}
type xmlField struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

func writeXML(w io.Writer, table string, records []Record) error {
	if table == "" {
		table = "dataset"
	}
	if _, err := fmt.Fprintf(w, "<%s>\n", table); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("  ", "  ")
	for _, r := range records {
		cols := make([]string, 0, len(r))
		for k := range r {
			cols = append(cols, k)
		}
		sort.Strings(cols)
		xr := xmlRecord{}
		for _, c := range cols {
			xr.Fields = append(xr.Fields, xmlField{Name: c, Value: fmt.Sprintf("%v", r[c])})
		}
		if err := enc.Encode(xr); err != nil {
			return err
		}
	}
	enc.Flush()
	_, err := fmt.Fprintf(w, "\n</%s>\n", table)
	return err
}

func writeSQL(w io.Writer, table string, records []Record) error {
	if table == "" {
		table = "synthetic_data"
	}
	cols := columnOrder(records)
	for _, r := range records {
		vals := make([]string, len(cols))
		for i, c := range cols {
			vals[i] = sqlLiteral(r[c])
		}
		line := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);\n",
			table, strings.Join(cols, ", "), strings.Join(vals, ", "))
		if _, err := io.WriteString(w, line); err != nil {
			return err
		}
	}
	return nil
}

func sqlLiteral(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return "NULL"
	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'"
	default:
		return fmt.Sprintf("%v", val)
	}
}

// writeBinary implements a tiny custom, length-prefixed binary container:
// [uint32 recordCount][per record: uint32 jsonLen][json bytes]. Documented
// as VOID's "custom Binary file" export target from the product spec.
func writeBinary(w io.Writer, records []Record) error {
	if err := binary.Write(w, binary.LittleEndian, uint32(len(records))); err != nil {
		return err
	}
	for _, r := range records {
		data, err := json.Marshal(r)
		if err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, uint32(len(data))); err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	return nil
}
