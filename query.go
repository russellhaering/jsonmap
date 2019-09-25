package jsonmap

import (
	"fmt"
	"strconv"
)

// Map an individual value between its struct and query representations
type QueryValueMapper interface {
	Decode(string) (interface{}, error)
	Encode(interface{}) (string, error)
}

// Map a slice of values between struct and query representation
type QueryParameterMapper interface {
	Decode([]string) (interface{}, error)
	Encode(interface{}) ([]string, error)
}

// SingletonParameterMapper wraps a QueryValueMapper to expose a Query
type SingletonParameterMapper struct {
	ValueMapper  QueryValueMapper
	IgnoreExtras bool
}

func (m *SingletonParameterMapper) Decode(in []string) (interface{}, error) {
	switch len(in) {
	case 0:
		return nil, nil

	case 1:
		return m.ValueMapper.Decode(in[0])
	default:
		if m.IgnoreExtras {
			return m.ValueMapper.Decode(in[0])
		}

		return nil, fmt.Errorf("received %d values, only expected 1", len(in))
	}
}

func (m *SingletonParameterMapper) Encode(in interface{}) ([]string, error) {
	out, err := m.ValueMapper.Encode(in)
	if err != nil {
		return nil, err
	}

	return []string{out}, nil
}

type RepeatedParameterMapper struct {
	ValueMapper  QueryValueMapper
	IgnoreExtras bool
}

func (m *RepeatedParameterMapper) Decode(in []string) (interface{}, error) {
	out := make([]interface{}, len(in))
	for i, val := range in {
		decoded, err := m.ValueMapper.Decode(val)
		if err != nil {
			return nil, err
		}

		out[i] = decoded
	}

	return out, nil
}

func (m *RepeatedParameterMapper) Encode(in interface{}) ([]string, error) {
	inSlice, ok := in.([]interface{})
	if !ok {
	}

	out := make([]string{}, len(in))
	out, err := m.ValueMapper.Encode(in)
	if err != nil {
		return nil, err
	}

	return []string{out}, nil
}

type StringQueryValueMapper struct {
	Validator Validator
}

func (m *StringQueryValueMapper) Encode(in interface{}) (string, error) {
	out, ok := in.(string)
	if !ok {
		return "", fmt.Errorf("invalid type")
	}

	return out, nil
}

func (m *StringQueryValueMapper) Decode(in string) (interface{}, error) {
	out, err := m.Validator.Validate(in)
	if err != nil {
		return nil, err
	}
	return out, nil
}



type IntegerValueMapper struct {
	Validator Validator
}

func (m *IntegerValueMapper) Encode(in interface{}) (string, error) {
	out, ok := in.(int)
	if !ok {
		return "", fmt.Errorf("invalid type")
	}

	return strconv.Itoa(out), nil
}

func (m *IntegerValueMapper) Decode(in string) (interface{}, error) {
	out, err := m.Validator.Validate(in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type MappedParameter struct {
	StructFieldName string
	ParameterName   string
	Mapper          QueryValueMapper
}

type QueryMap struct {
	UnderlyingType interface{}
	Parameters     []MappedParameter
}

var UserFilterMap = QueryMap{
	UserFilter{},
	[]MappedParameter{
		{
			StructFieldName: "Offset",
			ParameterName:   "offset",
			Mapper: SingletonParameterMapper{

			},
		},
	},
}
