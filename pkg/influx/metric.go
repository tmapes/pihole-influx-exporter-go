package influx

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
)

var (
	tagReplacer = strings.NewReplacer(
		",", "\\,",
		"=", "\\=",
		" ", "\\ ",
	)
	fieldReplacer = strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\",
	)
)

type Metric struct {
	Measurement string
	TimestampNS int64
	Tags        []tag
	Fields      []field
}

type tag struct {
	Name  string
	Value string
}

func (t tag) String() string {
	return fmt.Sprintf("%s=%s", t.Name, t.Value)
}

type field struct {
	Name  string
	Value string
}

func (f field) String() string {
	return fmt.Sprintf("%s=%s", f.Name, f.Value)
}

func NewMetric(measurement string, timestampNs int64) Metric {
	return Metric{
		Measurement: measurement,
		TimestampNS: timestampNs,
		Tags:        make([]tag, 0),
		Fields:      make([]field, 0),
	}
}

func (m *Metric) WithTag(name string, value string) error {
	if len(name) == 0 || len(value) == 0 {
		return errors.New("tag name and value must both be non-empty")
	} else if name[0] == '_' {
		return errors.New("tag name must not begin with an underscore")
	}

	m.Tags = append(m.Tags, tag{
		Name:  tagReplacer.Replace(name),
		Value: tagReplacer.Replace(value),
	})
	return nil
}

func (m *Metric) WithIntField(name string, value int) error {
	if len(name) == 0 {
		return errors.New("field name must be non-empty")
	} else if name[0] == '_' {
		return errors.New("field name must not begin with an underscore")
	}

	m.Fields = append(m.Fields, field{
		Name:  name,
		Value: fmt.Sprintf("%di", value),
	})

	return nil
}

func (m *Metric) WithFloatField(name string, value float64) error {
	if len(name) == 0 {
		return errors.New("field name must be non-empty")
	} else if name[0] == '_' {
		return errors.New("field name must not begin with an underscore")
	}

	m.Fields = append(m.Fields, field{
		Name:  name,
		Value: fmt.Sprintf("%f", value),
	})

	return nil
}

func (m *Metric) WithStringField(name string, value string) error {
	if len(name) == 0 {
		return errors.New("field name must be non-empty")
	} else if name[0] == '_' {
		return errors.New("field name must not begin with an underscore")
	}

	m.Fields = append(m.Fields, field{
		Name:  name,
		Value: fieldReplacer.Replace(value),
	})

	return nil
}

func (m *Metric) String() string {
	if len(m.Fields) == 0 {
		return ""
	}
	builder := strings.Builder{}
	builder.WriteString(m.Measurement)

	slices.SortFunc(m.Tags, func(a, b tag) int {
		return cmp.Compare(a.Name, b.Name)
	})
	for _, tag := range m.Tags {
		builder.WriteByte(',')
		builder.WriteString(tag.String())
	}

	builder.WriteByte(' ')

	slices.SortFunc(m.Fields, func(a, b field) int {
		return cmp.Compare(a.Name, b.Name)
	})
	lastIndex := len(m.Fields) - 1
	for i, f := range m.Fields {
		builder.WriteString(f.String())
		if i != lastIndex {
			builder.WriteByte(',')
		}
	}

	builder.WriteString(fmt.Sprintf(" %d", m.TimestampNS))

	return builder.String()
}
