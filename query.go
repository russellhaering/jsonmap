package jsonmap

import (
	"errors"
	"reflect"
	"strconv"
)

// This is the overarching struct used to transform structs into url params
// and vice versa
type QueryMap struct {
	UnderlyingType interface{}
	Parameters     []MappedParameter
}

// Taking a struct and turning it into a url param. The precise mechanisms of doing
// so are are defined in the individual MappedParameter
func (qm QueryMap) Encode(src interface{}, urlQuery map[string][]string) error {
	srcVal := reflect.ValueOf(src)
	for _, p := range qm.Parameters {
		field := srcVal.FieldByName(p.StructFieldName)
		strVal, err := p.Mapper.Encode(reflect.ValueOf(field))
		if err != nil {
			return errors.New("error in encoding struct")
		}

		urlQuery[p.ParameterName] = strVal
	}
	return nil
}

func (qm QueryMap) Decode(urlQuery map[string][]string, dst interface{}) error {
	if reflect.ValueOf(dst).Elem().Type() != reflect.TypeOf(qm.UnderlyingType) {
		return errors.New("attempting to decode into the wrong struct")
	}

	dstVal := reflect.ValueOf(dst).Elem()
	for _, param := range qm.Parameters {
		field := dstVal.FieldByName(param.StructFieldName)
		fieldVal, err := param.Mapper.Decode(urlQuery[param.ParameterName])
		if err != nil {
			return err
		}

		field.Set(reflect.ValueOf(fieldVal))
	}
	return nil
}

// MappedParameter corresponds to each field in a specific struct,
// it requires struct's name and the corresponding key value in the URL query
type MappedParameter struct {
	StructFieldName string
	ParameterName   string
	Mapper          QueryParameterMapper
}

// QueryParameterMapper defines how Query value and struct are to be transformed
// into each other. It is from a slice of strings, reflecting the structure of url.Values
// These can be specified by their type (whichever struct the Parameter mapper will be,
// and the restrictions defined on the type, defined by Validators slice below)
type QueryParameterMapper interface {
	Encode(interface{}) ([]string, error)
	Decode([]string) (interface{}, error)
}

// Examples of mappers
type StringQueryParameterMapper struct {
	Validators []func(string) bool
}

func (sqpm StringQueryParameterMapper) Decode(src []string) (interface{}, error) {
	if len(src) != 1 {
		return nil, errors.New("expected only one value")
	}

	str := 	src[0]
	for _, v := range sqpm.Validators {
		if !v(str) {
			return nil, errors.New("a validation test failed")
		}
	}
	return str, nil
}

func (sqpm StringQueryParameterMapper) Encode(src interface{}) ([]string, error) {
	s, ok := src.(string)
	if !ok {
		return nil, errors.New("improper type")
	}

	return []string{s}, nil
}

type IntQueryParameterMapper struct {
	Validators []func(int) bool
}

func (iqpm IntQueryParameterMapper) Decode(src []string) (interface{}, error) {
	if len(src) != 1 {
		return nil, errors.New("expected only one value")
	}

	num, err := strconv.Atoi(src[0])
	if err != nil {
		return nil, errors.New("param could not be converted to integer")
	}

	for _, v := range iqpm.Validators {
		if !v(num) {
			return nil, errors.New("a validation test failed")
		}
	}
	return num, nil
}

func (iqpm IntQueryParameterMapper) Encode(src interface{}) ([]string, error) {
	n, ok := src.(int)
	if !ok {
		return nil, errors.New("improper type")
	}
	return []string{strconv.Itoa(n)}, nil
}

type StrSliceQueryParameterMapper struct {
	Validators []func([]string) bool
	UnderlyingQueryParameterMapper QueryParameterMapper
}

func (sqpm StrSliceQueryParameterMapper) Decode(src []string) (interface{}, error) {
	for _, val := range sqpm.Validators {
		if !val(src) {
		}
	}

	var retVal []string
	// My brain has been sufficiently poisoned by this code.
	// There's probably a better way to do this, but this works and keeps QueryMap.Decode
	// ignorant of the internals of the fields
	for _, s := range src {
		v, err := sqpm.UnderlyingQueryParameterMapper.Decode([]string{s})
		if err != nil {
			return nil, err
		}
		retVal = append(retVal, v.(string))
	}
	return retVal, nil
}

func (sqpm StrSliceQueryParameterMapper) Encode(src interface{}) ([]string, error) {
	slice, ok := src.([]interface{})
	if !ok {
		return nil, errors.New("improper type")
	}

	var retSlice []string
	for _, v := range slice {
		s, err := sqpm.UnderlyingQueryParameterMapper.Encode(v)
		if err != nil {
			return nil, errors.New("error in encoding slice internals: " + err.Error())
		}
		retSlice = append(retSlice, s[0])
	}

	return retSlice, nil
}
