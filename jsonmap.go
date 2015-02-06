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

func (tm TypeMap) marshalField(field MappedField, srcField reflect.Value) (interface{}, error) {
	if field.Contains != nil {
		return field.Contains.Marshal(srcField)
	} else {
		return srcField.Interface(), nil
	}
}

func (tm TypeMap) marshalStruct(src reflect.Value) (json.Marshaler, error) {
	result := map[string]interface{}{}

	for _, field := range tm.Fields {
		srcField := src.FieldByName(field.StructFieldName)
		if !srcField.IsValid() {
			panic("No such underlying field: " + field.StructFieldName)
		}

		val, err := tm.marshalField(field, srcField)
		if err != nil {
			return nil, err
		}

		result[field.JSONFieldName] = val
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return russellRawMessage{data}, nil
}

func (tm TypeMap) marshalSlice(src reflect.Value) (json.Marshaler, error) {
	result := make([]interface{}, src.Len())

	for i := 0; i < src.Len(); i++ {
		data, err := tm.Marshal(src.Index(i))
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

func (tm TypeMap) Marshal(src reflect.Value) (json.Marshaler, error) {
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}

	if src.Kind() == reflect.Slice {
		return tm.marshalSlice(src)
	} else {
		return tm.marshalStruct(src)
	}
}

type TypeMapper struct {
	typeMaps map[reflect.Type]TypeMap
}

func NewTypeMapper(maps ...TypeMap) *TypeMapper {
	t := &TypeMapper{
		typeMaps: make(map[reflect.Type]TypeMap),
	}
	for _, m := range maps {
		t.typeMaps[reflect.TypeOf(m.UnderlyingType)] = m
	}
	return t
}

func (tm *TypeMapper) getTypeMap(obj interface{}) TypeMap {
	t := reflect.TypeOf(obj)

	// Iterate down through pointers and slices to get a useful type. Mostly this
	// doesn't seem useful, but we should at least support slices of pointers to
	// mappable types.
	for {
		if t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
			t = t.Elem()
			continue
		}
		break
	}

	m, ok := tm.typeMaps[t]

	if !ok {
		panic("no TypeMap registered for type: " + t.String())
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
