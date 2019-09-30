package jsonmap

import (
	"errors"
	"fmt"
	"net/http"
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
		strVal, err := p.Mapper.Encode(srcVal.FieldByName(p.StructFieldName))
		if err != nil {
			return errors.New("error in encoding struct: " + err.Error())
		}

		urlQuery[p.ParameterName] = strVal
	}
	return nil
}

// Taking a URL Query (or any string->[]string struct) and shoving it into the struct
// as specified by qm.UnderlyingType
func (qm QueryMap) Decode(urlQuery map[string][]string, dst interface{}) error {
	if reflect.ValueOf(dst).Elem().Type() != reflect.TypeOf(qm.UnderlyingType) {
		return fmt.Errorf("attempting to decode into mismatched struct: expected %s but got %s",
			reflect.TypeOf(qm.UnderlyingType),
			reflect.ValueOf(dst).Elem().Type(),
		)
	}

	dstVal := reflect.ValueOf(dst).Elem()
	for _, param := range qm.Parameters {
		field := dstVal.FieldByName(param.StructFieldName)
		fieldVal, err := param.Mapper.Decode(urlQuery[param.ParameterName])
		if err != nil {
			return fmt.Errorf("error ocurred while reading value: %s",
				err.Error(),
			)
		}

		field.Set(reflect.ValueOf(fieldVal))
	}
	return nil
}

// This ignores the case of parameter name in favor of the canonical format of
// http.Header
func (qm QueryMap) EncodeHeader(src interface{}, headers http.Header) error {
	srcVal := reflect.ValueOf(src)
	for _, p := range qm.Parameters {
		field := srcVal.FieldByName(p.StructFieldName)
		sliVal, err := p.Mapper.Encode(field)
		if err != nil {
			return errors.New("error in encoding struct: " + err.Error())
		}

		// Not using .Set() because it only allows strings and not slices
		headers[http.CanonicalHeaderKey(p.ParameterName)] = sliVal
	}
	return nil
}

func (qm QueryMap) DecodeHeader(headers http.Header, dst interface{}) error {
	if reflect.ValueOf(dst).Elem().Type() != reflect.TypeOf(qm.UnderlyingType) {
		return errors.New("attempting to decode into the wrong struct")
	}

	dstVal := reflect.ValueOf(dst).Elem()
	for _, param := range qm.Parameters {
		field := dstVal.FieldByName(param.StructFieldName)
		fieldVal, err := param.Mapper.Decode(headers[http.CanonicalHeaderKey(param.ParameterName)])
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
	Encode(reflect.Value) ([]string, error)
	Decode([]string) (interface{}, error)
}

// Examples of mappers
type StringQueryParameterMapper struct {
	Validators []func(string) bool
}

func (sqpm StringQueryParameterMapper) Decode(src []string) (interface{}, error) {
	if len(src) != 1 {
		return nil, fmt.Errorf("expected one value, but got %d", len(src))
	}

	str := src[0]
	for _, v := range sqpm.Validators {
		if !v(str) {
			return nil, errors.New("a validation test failed")
		}
	}
	return str, nil
}

func (sqpm StringQueryParameterMapper) Encode(src reflect.Value) ([]string, error) {
	if src.Kind() != reflect.String {
		return nil, fmt.Errorf("expected string but got: %s", src.Kind())
	}
	return []string{src.String()}, nil
}

// Does this need validators?
type BoolQueryParameterMapper struct{}

func (bqpm BoolQueryParameterMapper) Decode(src []string) (interface{}, error) {
	if len(src) != 1 {
		return nil, fmt.Errorf("expected one value, but got %d", len(src))
	}

	b, err := strconv.ParseBool(src[0])
	if err != nil {
		return nil, fmt.Errorf("could not parse into bool: %s", err.Error())
	}
	return b, nil
}

func (bqpm BoolQueryParameterMapper) Encode(src reflect.Value) ([]string, error) {
	if src.Kind() != reflect.Bool {
		return nil, fmt.Errorf("expected boolean but got: %s", src.Kind())
	}
	return []string{strconv.FormatBool(src.Bool())}, nil
}

type IntQueryParameterMapper struct {
	Validators []func(int) bool
}

func (iqpm IntQueryParameterMapper) Decode(src []string) (interface{}, error) {
	if len(src) != 1 {
		return nil, fmt.Errorf("expected one value, but got %d", len(src))
	}

	num, err := strconv.Atoi(src[0])
	if err != nil {
		return nil, fmt.Errorf("param could not be converted to integer: %s", err.Error())
	}

	for _, v := range iqpm.Validators {
		if !v(num) {
			return nil, errors.New("a validation test failed")
		}
	}
	return num, nil
}

func (iqpm IntQueryParameterMapper) Encode(src reflect.Value) ([]string, error) {
	if src.Kind() != reflect.Int {
		return nil, fmt.Errorf("expected int but got: %s", src.Kind())
	}
	return []string{strconv.FormatInt(src.Int(), 10)}, nil // Itoa doesn't take int64
}

type StrSliceQueryParameterMapper struct {
	Validators                     []func([]string) bool
	UnderlyingQueryParameterMapper QueryParameterMapper
}

func (sqpm StrSliceQueryParameterMapper) Decode(src []string) (interface{}, error) {
	for _, val := range sqpm.Validators {
		if !val(src) {
			return nil, errors.New("A validation test failed")
		}
	}

	var retVal []string
	// My brain has been sufficiently poisoned by this code.
	// There's probably a better way to do this, but this works and keeps QueryMap.Decode
	// ignorant of the internals of the fields
	for _, s := range src {
		v, err := sqpm.UnderlyingQueryParameterMapper.Decode([]string{s})
		if err != nil {
			return nil, fmt.Errorf("decoding a slice element failed: %s", err.Error())
		}
		retVal = append(retVal, v.(string))
	}
	return retVal, nil
}

func (sqpm StrSliceQueryParameterMapper) Encode(src reflect.Value) ([]string, error) {
	if src.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected slice but got: %s", src.Kind())
	}
	var retSlice []string
	for i := 0; i < src.Len(); i++ {
		s, err := sqpm.UnderlyingQueryParameterMapper.Encode(src.Index(i))
		if err != nil {
			return nil, errors.New("error in encoding slice internals: " + err.Error())
		}
		retSlice = append(retSlice, s[0])
	}

	return retSlice, nil
}
