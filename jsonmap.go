package jsonmap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rnd42/go-jsonpointer"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type Context interface{}

var EmptyContext Context

var (
	nullJSONValue  = []byte("null")
	nullRawMessage = RawMessage{nullJSONValue}
)

type FlattenedPathError struct {
	Path    string
	Message string
}

func (e *FlattenedPathError) String() string {
	return fmt.Sprintf("%s: %s\n", e.Path, e.Message)
}

func NewFlattenedPathError(path, message string) *FlattenedPathError {
	return &FlattenedPathError{
		Path:    path,
		Message: message,
	}
}

type MultiValidationError struct {
	NestedErrors []*FlattenedPathError
}

func (e *MultiValidationError) Errors() []*FlattenedPathError {
	return e.NestedErrors
}

func (e *MultiValidationError) Error() string {
	b := strings.Builder{}
	b.WriteString("Validation Errors: \n")
	for _, f := range e.NestedErrors {
		b.WriteString(f.String())
	}
	return b.String()
}

func (e *MultiValidationError) AddError(err *ValidationError, path ...string) {
	path = append(path, err.Field)
	pointer := jsonpointer.NewJSONPointerFromTokens(&path)
	if err.Message != "" {
		jsonpath := pointer.String()
		e.NestedErrors = append(e.NestedErrors, NewFlattenedPathError(jsonpath, err.Message))
	}
	for _, v := range err.NestedErrors {
		e.AddError(v, path...)
	}
}

type ValidationError struct {
	Field        string
	Message      string
	NestedErrors []*ValidationError
}

func (e *ValidationError) ErrorMessage() string {
	if e.Field != "" && e.Message != "" {
		return fmt.Sprintf("%s: %s\n", e.Field, e.Message)
	}
	return e.Message
}

func (e *ValidationError) Error() string {
	prefix := e.Field
	msg := e.ErrorMessage()
	for _, f := range e.NestedErrors {
		msg += prefix + f.ErrorMessage()
	}
	return msg
}

func (e *ValidationError) AddError(err *ValidationError) {
	e.NestedErrors = append(e.NestedErrors, err)
}

func (e *ValidationError) SetField(field string) {
	e.Field = field
}

func NewValidationErrorWithField(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

func (e *ValidationError) Flatten() *MultiValidationError {
	me := &MultiValidationError{}
	for _, v := range e.NestedErrors {
		me.AddError(v)
	}
	return me
}

func NewValidationError(reason string, a ...interface{}) *ValidationError {
	return &ValidationError{
		Message: fmt.Sprintf(reason, a...),
	}
}

type Validator interface {
	Validate(interface{}) (interface{}, error)
}

type TypeMap interface {
	Unmarshal(ctx Context, parent *reflect.Value, partial interface{}, dstValue reflect.Value) error
	Marshal(ctx Context, parent *reflect.Value, field reflect.Value) (json.Marshaler, error)
}

type RegisterableTypeMap interface {
	TypeMap
	GetUnderlyingType() reflect.Type
}

type MappedField struct {
	StructFieldName  string
	StructGetterName string
	JSONFieldName    string
	Contains         TypeMap
	Validator        Validator
	Optional         bool
	ReadOnly         bool
}

type StructMap struct {
	UnderlyingType interface{}
	Fields         []MappedField
}

type RawMessage struct {
	Data []byte
}

func (rm RawMessage) MarshalJSON() ([]byte, error) {
	return rm.Data, nil
}

func (sm StructMap) GetUnderlyingType() reflect.Type {
	return reflect.TypeOf(sm.UnderlyingType)
}

func (sm StructMap) Unmarshal(ctx Context, parent *reflect.Value, partial interface{}, dstValue reflect.Value) error {
	if partial == nil && (dstValue.Kind() == reflect.Interface || dstValue.Kind() == reflect.Ptr) {
		return nil
	}

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

	if dstValue.Kind() == reflect.Ptr {
		dstValue.Set(reflect.New(reflect.TypeOf(sm.UnderlyingType)))
		dstValue = dstValue.Elem()
	}

	errs := &ValidationError{}

	for _, field := range sm.Fields {
		if field.ReadOnly {
			continue
		}

		// TODO: Setters
		dstField := dstValue.FieldByName(field.StructFieldName)
		if !dstField.IsValid() {
			panic("no such underlying field: " + field.StructFieldName)
		}

		val, ok := data[field.JSONFieldName]
		if !ok {
			if field.Optional {
				continue
			} else {
				err := NewValidationErrorWithField(field.JSONFieldName, "missing required field")
				errs.AddError(err)
				continue
			}
		}

		if val == nil && field.Optional {
			continue
		}

		var err error

		if field.Contains != nil {
			err = field.Contains.Unmarshal(ctx, &dstValue, val, dstField)
		} else if field.Validator != nil {
			val, err = field.Validator.Validate(val)
			// Check reflect.ValueOf(val).IsValid() instead of err == nil if returning the invalid input in Validate
			if err == nil {
				dstField.Set(reflect.ValueOf(val))
			}
		} else {
			panic("Field must have Contains or Validator: " + field.JSONFieldName)
		}

		if err != nil {
			switch e := err.(type) {
			case *ValidationError:
				e.SetField(field.JSONFieldName)
				errs.AddError(e)
			default:
				ve := NewValidationErrorWithField(field.JSONFieldName, e.Error())
				errs.AddError(ve)
			}
		}
	}

	if len(errs.NestedErrors) != 0 {
		return errs
	}

	return nil
}

func (sm StructMap) marshalField(ctx Context, parent reflect.Value, field MappedField, srcField reflect.Value) ([]byte, error) {
	var val interface{}
	if field.Contains != nil {
		var err error
		val, err = field.Contains.Marshal(ctx, &parent, srcField)
		if err != nil {
			return nil, err
		}
	} else {
		val = srcField.Interface()
	}

	return json.Marshal(val)
}

func (sm StructMap) Marshal(ctx Context, parent *reflect.Value, src reflect.Value) (json.Marshaler, error) {
	buf := bytes.Buffer{}
	isNil := false

	// An Interface's Elem() returns a Ptr whose Elem() returns the actual value
	if src.Kind() == reflect.Interface {
		isNil = src.IsNil()
		src = src.Elem()
	}

	if src.Kind() == reflect.Ptr {
		isNil = src.IsNil()
		src = src.Elem()
	}

	if isNil {
		buf.Write(nullJSONValue)
	} else {
		expectedType := reflect.TypeOf(sm.UnderlyingType)
		if src.Type() != expectedType {
			panic("wrong type: " + src.Type().String() + ", expected: " + expectedType.String())
		}

		buf.WriteByte('{')

		for i, field := range sm.Fields {
			var srcField reflect.Value

			// TODO: Do validation ahead of time
			if field.StructFieldName != "" {
				srcField = src.FieldByName(field.StructFieldName)
				if !srcField.IsValid() {
					panic("no such underlying field: " + field.StructFieldName)
				}
			} else if field.StructGetterName != "" {
				// TODO: I'm not 100% sure if this works with methods that don't take a pointer
				srcGetter := src.Addr().MethodByName(field.StructGetterName)
				if !srcGetter.IsValid() {
					panic("no such underlying getter method: " + field.StructGetterName)
				}
				rets := srcGetter.Call([]reflect.Value{})
				if len(rets) != 2 {
					panic("invalid getter, should return (interface{}, error): " + field.StructGetterName)
				}
				if !rets[1].IsNil() {
					return nil, rets[1].Interface().(error)
				}
				srcField = rets[0]
			} else {
				panic("either StructFieldName or StructGetterName must be specified")
			}

			keybuf, err := json.Marshal(field.JSONFieldName)
			if err != nil {
				return nil, err
			}

			valbuf, err := sm.marshalField(ctx, src, field, srcField)
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
	}

	return RawMessage{buf.Bytes()}, nil
}

type SliceMap struct {
	Contains TypeMap
	MinLen   *int
	MaxLen   *int
}

func (sm SliceMap) Unmarshal(ctx Context, parent *reflect.Value, partial interface{}, dstValue reflect.Value) error {
	data, ok := partial.([]interface{})
	if !ok {
		return NewValidationError("expected a list")
	}

	err := sm.validateSliceWithinRange(data)
	if err != nil {
		return err
	}

	// Appending to a reflect.Value returns a new reflect.Value despite the
	// indirection. So we'll keep a reference to the original one, and Set()
	// it when we're done constructing the desired Value.
	result := dstValue

	elementType := dstValue.Type().Elem()

	errs := &ValidationError{}

	for i, val := range data {
		// Note: reflect.New() returns a pointer Value, so we have to take its
		// Elem() before putting it to use
		dstElem := reflect.New(elementType).Elem()

		err := sm.Contains.Unmarshal(ctx, &dstValue, val, dstElem)

		if err != nil {

			switch e := err.(type) {
			case *ValidationError:
				e.SetField(strconv.Itoa(i))
				errs.AddError(e)
			default:
				// This should never happen but just to be safe
				ve := NewValidationErrorWithField(strconv.Itoa(i), e.Error())
				errs.AddError(ve)
			}
			continue
		}

		result = reflect.Append(result, dstElem)
	}

	if len(errs.NestedErrors) != 0 {
		return errs
	}

	// Note: this actually works with a reflect.Value of a slice, even though it
	// wouldn't work with an actual slice because of the second level of
	// indirection.
	dstValue.Set(result)

	return nil
}

func (sm SliceMap) Marshal(ctx Context, parent *reflect.Value, src reflect.Value) (json.Marshaler, error) {
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}

	if src.IsNil() {
		return nullRawMessage, nil
	}

	result := make([]interface{}, src.Len())

	for i := 0; i < src.Len(); i++ {
		data, err := sm.Contains.Marshal(ctx, &src, src.Index(i))
		if err != nil {
			return nil, err
		}

		result[i] = data
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return RawMessage{data}, nil
}

func SliceOf(elem TypeMap) TypeMap {
	return SliceMap{
		Contains: elem,
	}
}

func SliceOfMax(elem TypeMap, max int) TypeMap {
	return SliceMap{
		Contains: elem,
		MaxLen:   &max,
	}
}

func SliceOfMin(elem TypeMap, min int) TypeMap {
	return SliceMap{
		Contains: elem,
		MinLen:   &min,
	}
}

func SliceOfRange(elem TypeMap, min, max int) TypeMap {
	return SliceMap{
		Contains: elem,
		MinLen:   &min,
		MaxLen:   &max,
	}
}

func (sm *SliceMap) validateSliceWithinRange(data []interface{}) error {
	if sm.MaxLen == nil && sm.MinLen == nil {
		return nil
	} else if sm.MaxLen == nil {
		if len(data) < *sm.MinLen {
			return NewValidationError("must have at least %d elements", *sm.MinLen)
		}
	} else if sm.MinLen == nil {
		if len(data) > *sm.MaxLen {
			return NewValidationError("must have at most %d elements", *sm.MaxLen)
		}
	} else if *sm.MaxLen == *sm.MinLen {
		if len(data) != *sm.MaxLen {
			return NewValidationError("must have %d elements", *sm.MaxLen)
		}
	} else if len(data) > *sm.MaxLen || len(data) < *sm.MinLen {
		return NewValidationError("must have between %d and %d elements", *sm.MinLen, *sm.MaxLen)
	}

	return nil
}

type MapMap struct {
	Contains TypeMap
}

func (mm MapMap) Unmarshal(ctx Context, parent *reflect.Value, partial interface{}, dstValue reflect.Value) error {
	data, ok := partial.(map[string]interface{})
	if !ok {
		return NewValidationError("expected a map")
	}

	errs := &ValidationError{}

	// Maps default to nil, so we need to make() one
	dstValue.Set(reflect.MakeMap(dstValue.Type()))

	elementType := dstValue.Type().Elem()

	for key, val := range data {
		// Note: reflect.New() returns a pointer Value, so we have to take its
		// Elem() before putting it to use
		dstElem := reflect.New(elementType).Elem()

		err := mm.Contains.Unmarshal(ctx, &dstValue, val, dstElem)

		if err != nil {
			switch e := err.(type) {
			case *ValidationError:
				e.SetField(key)
				errs.AddError(e)
			default:
				// This should never happen but just to be safe
				ne := NewValidationErrorWithField(key, e.Error())
				errs.AddError(ne)
			}
			continue
		}

		dstValue.SetMapIndex(reflect.ValueOf(key), dstElem)
	}
	if len(errs.NestedErrors) != 0 {
		return errs
	}

	return nil
}

func (mm MapMap) Marshal(ctx Context, parent *reflect.Value, src reflect.Value) (json.Marshaler, error) {
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}

	if src.IsNil() {
		return nullRawMessage, nil
	}

	result := make(map[string]interface{})
	keys := src.MapKeys()

	if src.Type().Key().Kind() != reflect.String {
		panic("key must be a string")
	}

	for _, key := range keys {
		data, err := mm.Contains.Marshal(ctx, &src, src.MapIndex(key))
		if err != nil {
			return nil, err
		}

		result[key.String()] = data
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return RawMessage{data}, nil
}

func MapOf(elem TypeMap) TypeMap {
	return &MapMap{
		Contains: elem,
	}
}

type toStringable interface {
	ToString() string
}

// This is a horrible hack of the go type system
type Discriminator struct {
	PropertyName string
	Mapping      map[string]TypeMap
}

func (vt *Discriminator) pickTypeMap(parent *reflect.Value) (TypeMap, error) {
	typeKeyField := parent.FieldByName(vt.PropertyName)
	if !typeKeyField.IsValid() {
		panic("no such underlying field: " + vt.PropertyName)
	}

	keyString := ""

	typeKey := typeKeyField.Interface()
	switch keyVal := typeKey.(type) {
	case string:
		keyString = keyVal
	case toStringable:
		keyString = keyVal.ToString()
	default:
		panic("cannot convert underlying field to string: " + typeKeyField.String())
	}

	typeMap, ok := vt.Mapping[keyString]

	if !ok {
		// NOTE: This error message isn't great because we don't have a way to know
		// the JSON field name uponw which we're switching.
		//TODO: include JSON field name uponw which we're switching to other error messages

		if keyString != "" {
			return nil, NewValidationError("invalid type identifier: '%s'", keyString)
		}

		if f, found := parent.Type().FieldByName(vt.PropertyName); found {
			jsonField := parseJsonTag(f)
			if jsonField != "" {
				return nil, NewValidationError("cannot validate, invalid input for '%s'", jsonField)
			}
		}

		return nil, NewValidationError("invalid type identifier")
	}

	return typeMap, nil
}

func (vt *Discriminator) Unmarshal(ctx Context, parent *reflect.Value, partial interface{}, dstValue reflect.Value) error {
	tm, err := vt.pickTypeMap(parent)
	if err != nil {
		return err
	}

	return tm.Unmarshal(ctx, parent, partial, dstValue)
}

func (vt *Discriminator) Marshal(ctx Context, parent *reflect.Value, src reflect.Value) (json.Marshaler, error) {
	if src.IsZero() {
		return nullRawMessage, nil
	}

	tm, err := vt.pickTypeMap(parent)
	if err != nil {
		panic("variable type serialization error: " + err.Error())
	}

	return tm.Marshal(ctx, parent, src)
}

func VariableType(switchOnFieldName string, types map[string]TypeMap) TypeMap {
	return &Discriminator{
		PropertyName: switchOnFieldName,
		Mapping:      types,
	}
}

type RenderInfo struct {
	Context Context
	Parent  interface{}
	Value   interface{}
}

type stringRenderer struct {
	template *template.Template
}

func (sr *stringRenderer) Unmarshal(ctx Context, parent *reflect.Value, partial interface{}, dstValue reflect.Value) error {
	return nil
}

func (sr *stringRenderer) Marshal(ctx Context, parent *reflect.Value, src reflect.Value) (json.Marshaler, error) {
	buf := bytes.Buffer{}
	err := sr.template.Execute(&buf, RenderInfo{
		Context: ctx,
		Parent:  parent.Interface(),
		Value:   src.Interface(),
	})

	if err != nil {
		return nil, err
	}

	marshalled, err := json.Marshal(buf.String())
	if err != nil {
		return nil, err
	}

	return RawMessage{marshalled}, nil
}

func StringRenderer(text string) *stringRenderer {
	return &stringRenderer{
		template: template.Must(template.New("").Parse(text)),
	}
}

type passthroughMarshaler struct{}

func (m *passthroughMarshaler) Marshal(ctx Context, parent *reflect.Value, field reflect.Value) (json.Marshaler, error) {
	data, err := json.Marshal(field.Interface())
	if err != nil {
		return nil, err
	}

	return RawMessage{data}, nil
}

type PrimitiveMap struct {
	passthroughMarshaler
	V Validator
}

func (m *PrimitiveMap) Unmarshal(ctx Context, parent *reflect.Value, partial interface{}, dstValue reflect.Value) error {
	val, err := m.V.Validate(partial)
	if err != nil {
		return err
	}

	if val != nil {
		dstValue.Set(reflect.ValueOf(val))
	}
	return nil
}

func NewPrimitiveMap(v Validator) TypeMap {
	return &PrimitiveMap{
		V: v,
	}
}

type TimeMap struct {
	passthroughMarshaler
}

func (m *TimeMap) Unmarshal(ctx Context, parent *reflect.Value, partial interface{}, dstValue reflect.Value) error {
	underlying := dstValue.Interface()
	if _, ok := underlying.(time.Time); !ok {
		panic("target field for jsonmap.Time() is not a time.Time")
	}

	tstring, ok := partial.(string)

	if !ok {
		return NewValidationError("not a string")
	}

	t, err := time.Parse(time.RFC3339, tstring)

	if err != nil {
		return NewValidationError("not a valid RFC 3339 time value")
	}

	dstValue.Set(reflect.ValueOf(t))

	return nil
}

func Time() TypeMap {
	return &TimeMap{}
}

type TypeMapper struct {
	typeMaps map[reflect.Type]TypeMap
}

func NewTypeMapper(maps ...RegisterableTypeMap) *TypeMapper {
	t := &TypeMapper{
		typeMaps: make(map[reflect.Type]TypeMap),
	}
	for _, m := range maps {
		t.typeMaps[m.GetUnderlyingType()] = m
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

func (tm *TypeMapper) Unmarshal(ctx Context, data []byte, dest interface{}) error {
	if reflect.TypeOf(dest).Kind() != reflect.Ptr || dest == nil {
		panic("cannot unmarshal to non-pointer")
	}
	m := tm.getTypeMap(dest)
	partial := map[string]interface{}{}

	err := json.Unmarshal(data, &partial)
	if err != nil {
		// We attempt to wrap json parse/unmarshal errors that can be caused by invalid input by
		// a validation error here. This is somewhat fragile and dependent on go's json impl.
		switch e := err.(type) {
		case *json.InvalidUnmarshalError:
			panic(e)
		case *json.SyntaxError:
			return NewValidationError(e.Error())
		case *json.UnmarshalTypeError:
			return NewValidationError("json: cannot unmarshal, not an object")
		default:
			// These are exported errors, but deprecated according to documentation.
			//case *json.InvalidUTF8Error:
			//case *json.UnmarshalFieldError:
			// These are exported errors, but only used for Marshal(). They are listed here for completeness.
			//case *json.MarshalerError:
			//case *json.UnsupportedTypeError:
			//case *json.UnsupportedValueError:
			return e
		}
	}
	err = m.Unmarshal(ctx, nil, partial, reflect.ValueOf(dest).Elem())
	if err != nil {
		if e, ok := err.(*ValidationError); ok {
			return e.Flatten()
		}
		return err
	}
	return nil
}

func (tm *TypeMapper) Marshal(ctx Context, src interface{}) ([]byte, error) {
	m := tm.getTypeMap(src)
	data, err := m.Marshal(ctx, nil, reflect.ValueOf(src))
	if err != nil {
		return nil, err
	}
	return data.MarshalJSON()
}

func (tm *TypeMapper) MarshalIndent(ctx Context, src interface{}, prefix, indent string) ([]byte, error) {
	// This is nuts, but equivalent to how json.MarshalIndent() works
	data, err := tm.Marshal(ctx, src)
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

// extracts the json field name from the field's json tag:
// `json:"bar,omitempty"` => "bar"
// `json:"bar"` => "bar"
// `json:"-"` => ""

func parseJsonTag(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	return strings.Split(tag, ",")[0]
}
