package jsonmap

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"time"
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
		fieldVal := srcVal.FieldByName(p.StructFieldName)

		if fieldVal.IsZero() && p.OmitEmpty {
			continue
		}

		strVal, err := p.Mapper.Encode(fieldVal)
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
	// First sanity check to ensure that the struct passed in matches
	// the struct the QueryMap was designed to handle
	if reflect.ValueOf(dst).Elem().Type() != reflect.TypeOf(qm.UnderlyingType) {
		return fmt.Errorf("attempting to decode into mismatched struct: expected %s but got %s",
			reflect.TypeOf(qm.UnderlyingType),
			reflect.ValueOf(dst).Elem().Type(),
		)
	}

	errs := &MultiValidationError{}
	dstVal := reflect.ValueOf(dst).Elem()
	for _, param := range qm.Parameters {
		paramValue := urlQuery[param.ParameterName]
		if len(paramValue) == 0 {
			continue
		}

		field := dstVal.FieldByName(param.StructFieldName)
		decodedParam, err := param.Mapper.Decode(paramValue)
		if err != nil {
			errs.AddError(NewValidationError("error ocurred while reading value (%s) into param %s: %s",
				urlQuery[param.ParameterName],
				param.StructFieldName,
				err.Error(),
			))
			continue
		}

		field.Set(reflect.ValueOf(decodedParam))
	}

	if len(errs.Errors()) == 0 {
		return nil
	}
	return errs
}

// This ignores the case of parameter name in favor of the canonical format of
// http.Header
func (qm QueryMap) EncodeHeader(src interface{}, headers http.Header) error {
	srcVal := reflect.ValueOf(src)

	for _, p := range qm.Parameters {
		fieldVal := srcVal.FieldByName(p.StructFieldName)

		if fieldVal.IsZero() && p.OmitEmpty {
			continue
		}

		sliVal, err := p.Mapper.Encode(fieldVal)
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

	// First sanity check to ensure that the struct passed in matches
	// the struct the QueryMap was designed to handle
	if reflect.ValueOf(dst).Elem().Type() != reflect.TypeOf(qm.UnderlyingType) {
		return fmt.Errorf("attempting to decode into mismatched struct: expected %s but got %s",
			reflect.TypeOf(qm.UnderlyingType),
			reflect.ValueOf(dst).Elem().Type(),
		)
	}

	errs := &MultiValidationError{}
	dstVal := reflect.ValueOf(dst).Elem()
	for _, param := range qm.Parameters {
		headerVal := headers[http.CanonicalHeaderKey(param.ParameterName)]
		if len(headerVal) == 0 {
			continue
		}

		field := dstVal.FieldByName(param.StructFieldName)
		decodedHeader, err := param.Mapper.Decode(headerVal)
		if err != nil {
			errs.AddError(NewValidationError("error ocurred while reading value (%s) into param %s: %s",
				headerVal,
				param.StructFieldName,
				err.Error(),
			))
			continue
		}

		field.Set(reflect.ValueOf(decodedHeader))
	}

	if len(errs.Errors()) == 0 {
		return nil
	}
	return errs
}

// MappedParameter corresponds to each field in a specific struct,
// it requires struct's name and the corresponding key value in the URL query
type MappedParameter struct {
	StructFieldName string
	ParameterName   string
	Mapper          QueryParameterMapper
	OmitEmpty       bool
}

// QueryParameterMapper defines how url.Values value ([]string) and struct are to be
// transformed into each other. It is from a slice of strings, reflecting the structure
// of url.Values. These can be specified by their type (whichever struct the Parameter
// mapper will be, and the restrictions defined on the type, defined by Validators slice
// below)
type QueryParameterMapper interface {
	Encode(reflect.Value) ([]string, error)
	Decode([]string) (interface{}, error)
}

// Examples of mappers
type StringQueryParameterMapper struct {
	Validators []func(string) bool
}

func (sqpm StringQueryParameterMapper) Decode(src []string) (interface{}, error) {
	// If more than one value is expected, use StrSliceQueryParameterMapper
	if len(src) != 1 {
		return nil, NewValidationError("expected only one value")
	}

	str := src[0]
	for _, v := range sqpm.Validators {
		if !v(str) {
			return nil, NewValidationError("a validation test failed")
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

// Some useful validators
func StringRangeValidator(min, max int) func(string) bool {
	return func(s string) bool {
		return min <= len(s) && len(s) <= max
	}
}

func StringRegexValidator(r *regexp.Regexp) func(string) bool {
	return func(s string) bool {
		return r.MatchString(s)
	}
}

// Probably doesn't need Validators
type BoolQueryParameterMapper struct{}

func (bqpm BoolQueryParameterMapper) Decode(src []string) (interface{}, error) {
	if len(src) != 1 {
		return nil, NewValidationError("expected only one value")
	}

	// Special case
	if src[0] == "" {
		return false, nil
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
	Validators []func(int64) bool
	BitSize    int
}

func (iqpm IntQueryParameterMapper) Decode(src []string) (interface{}, error) {
	if len(src) != 1 {
		return nil, NewValidationError("expected only one value")
	}

	num, err := strconv.ParseInt(src[0], 10, iqpm.BitSize)
	if err != nil {
		return nil, NewValidationError("param could not be converted to integer: %s",
			err.Error(),
		)
	}

	for _, v := range iqpm.Validators {
		if !v(num) {
			return nil, NewValidationError("a validation test failed")
		}
	}

	switch b := iqpm.BitSize; {
	case b == 0:
		return int(num), nil
	case b <= 8:
		return int8(num), nil
	case b <= 16:
		return int16(num), nil
	case b <= 32:
		return int32(num), nil
	}

	return num, nil
}

func (iqpm IntQueryParameterMapper) Encode(src reflect.Value) ([]string, error) {
	switch src.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []string{strconv.FormatInt(src.Int(), 10)}, nil
	default:
		return nil, fmt.Errorf("expected int-type but got: %s", src.Kind())
	}
}

type UintQueryParameterMapper struct {
	Validators []func(uint64) bool
	BitSize    int
}

func (uqpm UintQueryParameterMapper) Decode(src []string) (interface{}, error) {
	if len(src) != 1 {
		return nil, NewValidationError("expected only one value")
	}

	num, err := strconv.ParseUint(src[0], 10, uqpm.BitSize)
	if err != nil {
		return nil, NewValidationError("param could not be converted to integer: %s",
			err.Error(),
		)
	}

	for _, v := range uqpm.Validators {
		if !v(num) {
			return nil, NewValidationError("a validation test failed")
		}
	}

	switch b := uqpm.BitSize; {
	case b == 0:
		return uint(num), nil
	case b <= 8:
		return uint8(num), nil
	case b <= 16:
		return uint16(num), nil
	case b <= 32:
		return uint32(num), nil
	}

	return num, nil
}

func (uqpm UintQueryParameterMapper) Encode(src reflect.Value) ([]string, error) {
	switch src.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []string{strconv.FormatUint(src.Uint(), 10)}, nil
	default:
		return nil, fmt.Errorf("expected uint-type but got: %s", src.Kind())
	}
}

type TimeQueryParameterMapper struct {
	Validators []func(time.Time) bool
}

func (tqpm TimeQueryParameterMapper) Decode(src []string) (interface{}, error) {
	if len(src) != 1 {
		return nil, NewValidationError("expected one value, but got %d", len(src))
	}

	t := time.Time{}
	err := t.UnmarshalText([]byte(src[0]))
	if err != nil {
		return nil, NewValidationError("param could not be marshalled to time.Time: %s", err.Error())
	}

	for _, v := range tqpm.Validators {
		if !v(t) {
			return nil, NewValidationError("a validation test failed")
		}
	}
	return t, nil
}

func (tqpm TimeQueryParameterMapper) Encode(src reflect.Value) ([]string, error) {
	if src.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct but got: %s", src.Kind())
	}
	if src.Type() != reflect.TypeOf(time.Time{}) {
		return nil, fmt.Errorf("expected time.Time but got: %s", src.Type())
	}

	b, err := src.Interface().(time.Time).MarshalText()
	if err != nil {
		return nil, err
	}

	return []string{string(b)}, nil
}

type StrSliceQueryParameterMapper struct {
	Validators                     []func([]string) bool
	UnderlyingQueryParameterMapper QueryParameterMapper
}

func (sqpm StrSliceQueryParameterMapper) Decode(src []string) (interface{}, error) {
	for _, val := range sqpm.Validators {
		if !val(src) {
			return nil, NewValidationError("A validation test failed")
		}
	}

	var retVal []string
	// My brain has been sufficiently poisoned by this code.
	// There's probably a better way to do this, but this works and keeps QueryMap.Decode
	// ignorant of the internals of the fields
	for _, s := range src {
		v, err := sqpm.UnderlyingQueryParameterMapper.Decode([]string{s})
		if err != nil {
			return nil, NewValidationError("decoding a slice element failed: %s", err.Error())
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

type StrPointerQueryParameterMapper struct {
	UnderlyingQueryParameterMapper QueryParameterMapper
}

func (pqpm StrPointerQueryParameterMapper) Decode(src []string) (interface{}, error) {
	if len(src) != 1 {
		return nil, NewValidationError("expected one value, but got %d", len(src))
	}
	v, err := pqpm.UnderlyingQueryParameterMapper.Decode(src)
	if err != nil {
		return nil, NewValidationError("error occurred while decoding struct")
	}
	v2 := v.(string)
	return &v2, nil
}

func (pqpm StrPointerQueryParameterMapper) Encode(src reflect.Value) ([]string, error) {
	if src.Type() != reflect.PtrTo(reflect.TypeOf("")) {
		return nil, fmt.Errorf("expected pointer but got: %s", src.Kind())
	}
	return []string{src.Elem().String()}, nil
}
