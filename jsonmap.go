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

func (tm TypeMap) Marshal(src reflect.Value) (json.Marshaler, error) {
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}
	result := map[string]interface{}{}

	for _, field := range tm.Fields {
		srcField := src.FieldByName(field.StructFieldName)
		if !srcField.IsValid() {
			panic("No such underlying field: " + field.StructFieldName)
		}

		if field.Contains != nil {
			val, err := field.Contains.Marshal(srcField)
			if err != nil {
				return nil, err
			}
			result[field.JSONFieldName] = val
		} else {
			result[field.JSONFieldName] = srcField.Interface()
		}

	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return russellRawMessage{data}, nil
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
	if reflect.TypeOf(obj).Kind() != reflect.Ptr {
		panic("dst is not a pointer")
	}

	t := reflect.TypeOf(obj).Elem()
	m, ok := tm.typeMaps[t]

	if !ok {
		panic("no TypeMap registered for type: " + t.String())
	}

	return m
}

func (tm *TypeMapper) Unmarshal(data []byte, dest interface{}) error {
	m := tm.getTypeMap(dest)
	partial := map[string]interface{}{}

	err := json.Unmarshal(data, &partial)
	if err != nil {
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
