package jsonmap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

type ValidationError struct {
	reason string
	rpath  []string
}

func NewValidationError(reason string, a ...interface{}) *ValidationError {
	return &ValidationError{
		reason: fmt.Sprintf(reason, a...),
		rpath:  make([]string, 0, 2),
	}
}

func (e *ValidationError) Error() string {
	msg := e.reason
	for _, seg := range e.rpath {
		msg = seg + ": " + msg
	}
	return "validation error: " + msg
}

func (e *ValidationError) PushIndex(idx int) {
	e.rpath = append(e.rpath, fmt.Sprintf("index %d", idx))
}

func (e *ValidationError) PushKey(key string) {
	e.rpath = append(e.rpath, fmt.Sprintf("'%s'", key))
}

type Validator interface {
	Validate(interface{}) (interface{}, error)
}

type TypeMap interface {
	Unmarshal(parent *reflect.Value, partial interface{}, dstValue reflect.Value) error
	Marshal(parent *reflect.Value, field reflect.Value) (json.Marshaler, error)
}

type MappedField struct {
	StructFieldName string
	JSONFieldName   string
	Contains        TypeMap
	Validator       Validator
	Optional        bool
	ReadOnly        bool
}

type StructMap struct {
	UnderlyingType interface{}
	Fields         []MappedField
}

type russellRawMessage struct {
	Data []byte
}

func (rm russellRawMessage) MarshalJSON() ([]byte, error) {
	return rm.Data, nil
}

func (sm StructMap) Unmarshal(parent *reflect.Value, partial interface{}, dstValue reflect.Value) error {
	data, ok := partial.(map[string]interface{})
	if !ok {
		return NewValidationError("expected an object")
	}

	// In order to unmarshal into an interface{} we need to allocate an actual
	// instance of this type of struct, and set the interface{} to point to the
	// value.
	if dstValue.Kind() == reflect.Interface {
		dstValue.Set(reflect.New(reflect.TypeOf(sm.UnderlyingType)))
		dstValue = dstValue.Elem().Elem()
	}

	for _, field := range sm.Fields {
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
			err = field.Contains.Unmarshal(&dstValue, val, dstField)
		} else {
			val, err = field.Validator.Validate(val)
			if err == nil {
				dstField.Set(reflect.ValueOf(val))
			}
		}

		if err != nil {
			if ve, ok := err.(*ValidationError); ok {
				ve.PushKey(field.JSONFieldName)
			}
			return err
		}
	}

	return nil
}

func (sm StructMap) marshalField(parent reflect.Value, field MappedField, srcField reflect.Value) ([]byte, error) {
	var val interface{}
	if field.Contains != nil {
		var err error
		val, err = field.Contains.Marshal(&parent, srcField)
		if err != nil {
			return nil, err
		}
	} else {
		val = srcField.Interface()
	}

	return json.Marshal(val)
}

func (sm StructMap) Marshal(parent *reflect.Value, src reflect.Value) (json.Marshaler, error) {
	// An Interface's Elem() returns a Ptr whose Elem() returns the actual value
	if src.Kind() == reflect.Interface {
		src = src.Elem()
	}
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}

	expectedType := reflect.TypeOf(sm.UnderlyingType)
	if src.Type() != expectedType {
		panic("wrong type: " + src.Type().String() + ", expected: " + expectedType.String())
	}

	buf := bytes.Buffer{}
	buf.WriteByte('{')

	for i, field := range sm.Fields {
		srcField := src.FieldByName(field.StructFieldName)
		if !srcField.IsValid() {
			panic("No such underlying field: " + field.StructFieldName)
		}

		keybuf, err := json.Marshal(field.JSONFieldName)
		if err != nil {
			return nil, err
		}

		valbuf, err := sm.marshalField(src, field, srcField)
		if err != nil {
			return nil, err
		}

		buf.Write(keybuf)
		buf.WriteByte(':')
		buf.Write(valbuf)

		if i != len(sm.Fields)-1 {
			buf.WriteByte(',')
		}
	}

	buf.WriteByte('}')

	return russellRawMessage{buf.Bytes()}, nil
}

type SliceMap struct {
	Contains TypeMap
}

func (sm SliceMap) Unmarshal(parent *reflect.Value, partial interface{}, dstValue reflect.Value) error {
	data, ok := partial.([]interface{})
	if !ok {
		return NewValidationError("expected a list")
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

		err := sm.Contains.Unmarshal(&dstValue, val, dstElem)

		if err != nil {
			if ve, ok := err.(*ValidationError); ok {
				ve.PushIndex(i)
			}
			return err
		}

		result = reflect.Append(result, dstElem)
	}

	// Note: this actually works with a reflect.Value of a slice, even though it
	// wouldn't work with an actual slice because of the second level of
	// indirection.
	dstValue.Set(result)

	return nil
}

func (sm SliceMap) Marshal(parent *reflect.Value, src reflect.Value) (json.Marshaler, error) {
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}

	result := make([]interface{}, src.Len())

	for i := 0; i < src.Len(); i++ {
		data, err := sm.Contains.Marshal(&src, src.Index(i))
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

func SliceOf(elem TypeMap) TypeMap {
	return SliceMap{
		Contains: elem,
	}
}

// This is a horrible hack of the go type system
type variableType struct {
	switchOnFieldName string
	types             map[string]TypeMap
}

func (vt *variableType) pickTypeMap(parent *reflect.Value) (TypeMap, error) {
	typeKeyField := parent.FieldByName(vt.switchOnFieldName)
	if !typeKeyField.IsValid() {
		panic("No such underlying field: " + vt.switchOnFieldName)
	}

	typeKey := typeKeyField.String()
	typeMap, ok := vt.types[typeKey]

	if !ok {
		return nil, NewValidationError("unexpected value of '%s': %s", vt.switchOnFieldName, typeKey)
	}

	return typeMap, nil
}

func (vt *variableType) Unmarshal(parent *reflect.Value, partial interface{}, dstValue reflect.Value) error {
	tm, err := vt.pickTypeMap(parent)
	if err != nil {
		return err
	}

	return tm.Unmarshal(parent, partial, dstValue)
}

func (vt *variableType) Marshal(parent *reflect.Value, src reflect.Value) (json.Marshaler, error) {
	tm, err := vt.pickTypeMap(parent)
	if err != nil {
		return nil, err
	}

	return tm.Marshal(parent, src)
}

func VariableType(switchOnFieldName string, types map[string]TypeMap) TypeMap {
	return &variableType{
		switchOnFieldName: switchOnFieldName,
		types:             types,
	}
}

type TypeMapper struct {
	typeMaps map[reflect.Type]TypeMap
}

func NewTypeMapper(maps ...StructMap) *TypeMapper {
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

	return m.Unmarshal(nil, partial, reflect.ValueOf(dest).Elem())
}

func (tm *TypeMapper) Marshal(src interface{}) ([]byte, error) {
	m := tm.getTypeMap(src)
	data, err := m.Marshal(nil, reflect.ValueOf(src))
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
