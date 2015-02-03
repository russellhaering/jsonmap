package jsonmap

import (
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

type Unmarshaler interface {
	Unmarshal(partial interface{}, dstValue reflect.Value) error
}

type MappedField struct {
	StructFieldName string
	JSONFieldName   string
	Contains        Unmarshaler
	Validator       Validator
	Optional        bool
}

type TypeMap struct {
	UnderlyingType interface{}
	Fields         []MappedField
}

func (tm TypeMap) Unmarshal(partial interface{}, dstValue reflect.Value) error {
	data, ok := partial.(map[string]interface{})
	if !ok {
		return NewValidationError("expected a JSON object")
	}

	for _, field := range tm.Fields {
		dstField := dstValue.FieldByName(field.StructFieldName)
		if !dstField.IsValid() {
			panic("No such underlying field: " + field.StructFieldName)
		}

		val, ok := data[field.JSONFieldName]
		if !field.Optional && !ok {
			return NewValidationError("missing required field: %s", field.JSONFieldName)
		}

		if field.Contains != nil {
			return field.Contains.Unmarshal(val, dstField)
		} else {
			val, err := field.Validator.Validate(val)
			if err != nil {
				if ve, ok := err.(*ValidationError); ok {
					return NewValidationError("error validating field '%s': %s", field.JSONFieldName, ve.Error())
				} else {
					return err
				}
			}

			dstField.Set(reflect.ValueOf(val))
		}
	}

	return nil
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

func (tm *TypeMapper) Unmarshal(data []byte, dest interface{}) error {
	if reflect.TypeOf(dest).Kind() != reflect.Ptr {
		panic("dst is not a pointer")
	}

	t := reflect.TypeOf(dest).Elem()
	m, ok := tm.typeMaps[t]

	if !ok {
		panic("no TypeMap registered for type: " + t.String())
	}

	partial := map[string]interface{}{}

	err := json.Unmarshal(data, &partial)
	if err != nil {
		return err
	}

	return m.Unmarshal(partial, reflect.ValueOf(dest).Elem())
}
