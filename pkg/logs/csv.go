package logs

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
)

type (
	CSVConfig struct {
		Delimiter        rune                             `yaml:"delimiter"`
		TrimLeadingSpace bool                             `yaml:"trim_leading_space"`
		Format           string                           `yaml:"format"`
		CheckField       func(string) (string, int, bool) `yaml:"-"`
	}

	CSVParser struct {
		config CSVConfig
		reader *csv.Reader
		format *csvFormat
	}

	csvFormat struct {
		raw          string
		maxIndex     int
		fieldIndexes map[string]int
	}
)

func NewCSVParser(config CSVConfig, in io.Reader) (*CSVParser, error) {
	if config.Format == "" {
		return nil, errors.New("empty csv format")
	}

	format, err := newCSVFormat(config)
	if err != nil || len(format.fieldIndexes) == 0 {
		return nil, fmt.Errorf("bad csv format: %v", err)
	}

	p := &CSVParser{
		config: config,
		reader: newCSVReader(in, config),
		format: format,
	}
	return p, nil
}

func (p *CSVParser) ReadLine(line LogLine) error {
	record, err := p.reader.Read()
	if err != nil {
		return handleCSVReadError(err)
	}
	return p.format.parse(record, line)
}

func (p *CSVParser) Parse(row []byte, line LogLine) error {
	r := newCSVReader(bytes.NewBuffer(row), p.config)
	record, err := r.Read()
	if err != nil {
		return handleCSVReadError(err)
	}
	return p.format.parse(record, line)
}

func (p CSVParser) Info() string {
	return fmt.Sprintf("csv: %s", p.format.raw)
}

func (f *csvFormat) parse(record []string, line LogLine) error {
	if len(record) <= f.maxIndex {
		return &ParseError{msg: "csv parse: unmatched line"}
	}

	for field, idx := range f.fieldIndexes {
		if err := line.Assign(field, record[idx]); err != nil {
			return &ParseError{msg: fmt.Sprintf("csv parse: %v", err), err: err}
		}
	}
	return nil
}

func newCSVReader(in io.Reader, config CSVConfig) *csv.Reader {
	r := csv.NewReader(in)
	r.Comma = config.Delimiter
	r.TrimLeadingSpace = config.TrimLeadingSpace
	r.ReuseRecord = true
	r.FieldsPerRecord = -1
	return r
}

func newCSVFormat(config CSVConfig) (*csvFormat, error) {
	r := csv.NewReader(strings.NewReader(config.Format))
	r.Comma = config.Delimiter
	r.TrimLeadingSpace = config.TrimLeadingSpace

	fields, err := r.Read()
	if err != nil {
		return nil, err
	}

	check := checkCSVFormatField
	if config.CheckField != nil {
		check = config.CheckField
	}

	fieldIndexes := make(map[string]int)
	var info string
	var max int
	var offset int

	for i, field := range fields {
		field = strings.Trim(field, `"`)

		name, addOffset, valid := check(field)
		offset += addOffset
		if !valid {
			continue
		}

		idx := i + offset
		_, ok := fieldIndexes[name]
		if ok {
			return nil, fmt.Errorf("duplicate field: %s", name)
		}
		fieldIndexes[name] = idx
		info += fmt.Sprintf(" %d:%s", idx, name)
		if max < idx {
			max = idx
		}
	}

	format := &csvFormat{
		raw:          config.Format,
		maxIndex:     max,
		fieldIndexes: fieldIndexes,
	}

	return format, nil
}

func handleCSVReadError(err error) error {
	if isCSVParseError(err) {
		return &ParseError{msg: fmt.Sprintf("csv parse: %v", err), err: err}
	}
	return err
}

func isCSVParseError(err error) bool {
	return errors.Is(err, csv.ErrBareQuote) || errors.Is(err, csv.ErrFieldCount) || errors.Is(err, csv.ErrQuote)
}

func checkCSVFormatField(name string) (newName string, offset int, valid bool) {
	if len(name) < 2 || !strings.HasPrefix(name, "$") {
		return "", 0, false
	}
	return name, 0, true
}
