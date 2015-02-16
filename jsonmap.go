package jsonmap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

type ValidationError struct {
	reason string
}

func NewValidationError(reason string, a ...interface{}) *ValidationError {
	return &ValidationError{
		reason: fmt.Sprintf(reason, a...),
	}
}

func (e *ValidationError) Error() string {
	return e.reason
}

type Validator interface {
	Validate(interface{}) (interface{}, error)
}

type Encoder interface {
	Unmarshal(partial interface{}, dstValue reflect.Value) error
	Marshal(reflect.Value) (json.Marshaler, error)
}

type MappedField struct {
	StructFieldName string
	JSONFieldName   string
	Contains        Encoder
	Validator       Validator
	Optional        bool
	ReadOnly        bool
}

type TypeMap struct {
	UnderlyingType interface{}
	Fields         []MappedField
}

type russellRawMessage struct {
	Data []byte
}

func (rm russellRawMessage) MarshalJSON() ([]byte, error) {
	return rm.Data, nil
}

func (tm TypeMap) Unmarshal(partial interface{}, dstValue reflect.Value) error {
	data, ok := partial.(map[string]interface{})
	if !ok {
		return NewValidationError("expected a JSON object")
	}

	for _, field := range tm.Fields {
		if field.ReadOnly {
			continue
		}

		dstField := dstValue.FieldByName(field.StructFieldName)
		if !dstField.IsValid() {
			panic("No such underlying field: " + field.StructFieldName)
		}

		val, ok := data[field.JSONFieldName]
		if !ok {
			if field.Optional {
				continue
			} else {
				return NewValidationError("missing required field: %s", field.JSONFieldName)
			}
		}

		var err error

		if field.Contains != nil {
			err = field.Contains.Unmarshal(val, dstField)
		} else {
			val, err = field.Validator.Validate(val)
			if err == nil {
				dstField.Set(reflect.ValueOf(val))
			}
		}

		if err != nil {
			if ve, ok := err.(*ValidationError); ok {
				return NewValidationError("error validating field '%s': %s", field.JSONFieldName, ve.Error())
			} else {
				return err
			}
		}
	}

	return nil
}

func (tm TypeMap) marshalField(field MappedField, srcField reflect.Value) ([]byte, error) {
	var val interface{}
	if field.Contains != nil {
		var err error
		val, err = field.Contains.Marshal(srcField)
		if err != nil {
			return nil, err
		}
	} else {
		val = srcField.Interface()
	}

	return json.Marshal(val)
}

func (tm TypeMap) Marshal(src reflect.Value) (json.Marshaler, error) {
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}

	buf := bytes.Buffer{}
	buf.WriteByte('{')

	for i, field := range tm.Fields {
		srcField := src.FieldByName(field.StructFieldName)
		if !srcField.IsValid() {
			panic("No such underlying field: " + field.StructFieldName)
		}

		keybuf, err := json.Marshal(field.JSONFieldName)
		if err != nil {
			return nil, err
		}

		valbuf, err := tm.marshalField(field, srcField)
		if err != nil {
			return nil, err
		}

		buf.Write(keybuf)
		buf.WriteByte(':')
		buf.Write(valbuf)

		if i != len(tm.Fields)-1 {
			buf.WriteByte(',')
		}
	}

	buf.WriteByte('}')

	return russellRawMessage{buf.Bytes()}, nil
}

type SliceTypeMap struct {
	Contains Encoder
}

func (tm SliceTypeMap) Unmarshal(partial interface{}, dstValue reflect.Value) error {
	data, ok := partial.([]interface{})
	if !ok {
		return NewValidationError("expected a JSON list")
	}

	// Appending to a reflect.Value returns a new reflect.Value despite the
	// indirection. So we'll keep a reference to the original one, and Set()
	// it when we're done constructing the desired Value.
	result := dstValue

	elementType := dstValue.Type().Elem()

	for i, val := range data {
		// Note: reflect.New() returns a pointer Value, so we have to take its
		// Elem() before putting it to use
		dstElem := reflect.New(elementType).Elem()

		err := tm.Contains.Unmarshal(val, dstElem)

		if err != nil {
			if ve, ok := err.(*ValidationError); ok {
				return NewValidationError("error validating index %d: %s", i, ve.Error())
			} else {
				return err
			}
		}

		result = reflect.Append(result, dstElem)
	}

	// Note: this actually works with a reflect.Value of a slice, even though it
	// wouldn't work with an actual slice because of the second level of
	// indirection.
	dstValue.Set(result)

	return nil
}

func (tm SliceTypeMap) Marshal(src reflect.Value) (json.Marshaler, error) {
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}

	result := make([]interface{}, src.Len())

	for i := 0; i < src.Len(); i++ {
		data, err := tm.Contains.Marshal(src.Index(i))
		if err != nil {
			return nil, err
		}

		result[i] = data
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return russellRawMessage{data}, nil
}

func SliceOf(elem Encoder) Encoder {
	return SliceTypeMap{
		Contains: elem,
	}
}

type TypeMapper struct {
	typeMaps map[reflect.Type]Encoder
}

func NewTypeMapper(maps ...TypeMap) *TypeMapper {
	t := &TypeMapper{
		typeMaps: make(map[reflect.Type]Encoder),
	}
	for _, m := range maps {
		t.typeMaps[reflect.TypeOf(m.UnderlyingType)] = m
	}
	return t
}

func (tm *TypeMapper) getTypeMap(obj interface{}) Encoder {
	t := reflect.TypeOf(obj)
	isSlice := false

	if t.Kind() == reflect.Slice {
		isSlice = true
		t = t.Elem()
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	m, ok := tm.typeMaps[t]

	if !ok {
		panic("no TypeMap registered for type: " + t.String())
	}

	if isSlice {
		m = SliceOf(m)
	}

	return m
}

func (tm *TypeMapper) Unmarshal(data []byte, dest interface{}) error {
	if reflect.TypeOf(dest).Kind() != reflect.Ptr {
		panic("cannot unmarshal to non-pointer")
	}
	m := tm.getTypeMap(dest)
	partial := map[string]interface{}{}

	err := json.Unmarshal(data, &partial)
	if err != nil {
		if _, ok := err.(*json.SyntaxError); ok {
			err = NewValidationError(err.Error())
		}
		return err
	}

	return m.Unmarshal(partial, reflect.ValueOf(dest).Elem())
}

func (tm *TypeMapper) Marshal(src interface{}) ([]byte, error) {
	m := tm.getTypeMap(src)
	data, err := m.Marshal(reflect.ValueOf(src))
	if err != nil {
		return nil, err
	}
	return data.MarshalJSON()
}

func (tm *TypeMapper) MarshalIndent(src interface{}, prefix, indent string) ([]byte, error) {
	// This is nuts, but equivalent to how json.MarshalIndent() works
	data, err := tm.Marshal(src)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}

	err = json.Indent(buf, data, prefix, indent)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
