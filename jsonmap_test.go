package jsonmap

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type brokenValidator struct{}

func (v brokenValidator) Validate(interface{}) (interface{}, error) {
	return nil, errors.New("this should be a ValidationError")
}

type InnerThing struct {
	Foo   string
	AnInt int
	ABool bool
}

type OuterThing struct {
	InnerThing InnerThing
}

type OuterPointerThing struct {
	InnerThing *InnerThing
}

type OuterInterfaceThing struct {
	InnerThing interface{}
}

type OuterSliceThing struct {
	InnerThings []InnerThing
}

type OuterPointerSliceThing struct {
	InnerThings []*InnerThing
}

type OuterPointerToSliceThing struct {
	InnerThings *[]InnerThing
}

type OtherInnerThing struct {
	Bar string
}

type OuterVariableThing struct {
	InnerType  string
	InnerValue interface{}
}

type OtherOuterVariableThing OuterVariableThing

type ReadOnlyThing struct {
	PrimaryKey string
}

type UnregisteredThing struct {
}

type TypoedThing struct {
	Correct bool
}

type BrokenThing struct {
	Invalid string
}

type TemplatableThing struct {
	SomeField string
}

type NonMarshalableType struct{}

func (t NonMarshalableType) MarshalJSON() ([]byte, error) {
	return nil, errors.New("oops")
}

type InnerNonMarshalableThing struct {
	Oops NonMarshalableType
}

type OuterNonMarshalableThing struct {
	InnerThing InnerNonMarshalableThing
}

type ThingWithSliceOfPrimitives struct {
	Strings []string
}

type ThingWithMapOfInterfaces struct {
	Interfaces map[string]interface{}
}

type ThingWithTime struct {
	HappenedAt time.Time
}

type ThingWithEnumerableInterface struct {
	ThanksGo interface{}
}

var InnerThingTypeMap = StructMap{
	InnerThing{},
	[]MappedField{
		{
			StructFieldName: "Foo",
			JSONFieldName:   "foo",
			Validator:       String(1, 12),
			Optional:        true,
		},
		{
			StructFieldName: "AnInt",
			JSONFieldName:   "an_int",
			Validator:       Integer(0, 10),
			Optional:        true,
		},
		{
			StructFieldName: "ABool",
			JSONFieldName:   "a_bool",
			Validator:       Boolean(),
			Optional:        true,
		},
	},
}

var OuterThingTypeMap = StructMap{
	OuterThing{},
	[]MappedField{
		{
			StructFieldName: "InnerThing",
			JSONFieldName:   "inner_thing",
			Contains:        InnerThingTypeMap,
		},
	},
}

var OuterPointerThingTypeMap = StructMap{
	OuterPointerThing{},
	[]MappedField{
		{
			StructFieldName: "InnerThing",
			JSONFieldName:   "inner_thing",
			Contains:        InnerThingTypeMap,
		},
	},
}

var OuterInterfaceThingTypeMap = StructMap{
	OuterInterfaceThing{},
	[]MappedField{
		{
			StructFieldName: "InnerThing",
			JSONFieldName:   "inner_thing",
			Contains:        InnerThingTypeMap,
		},
	},
}

var OuterSliceThingTypeMap = StructMap{
	OuterSliceThing{},
	[]MappedField{
		{
			StructFieldName: "InnerThings",
			JSONFieldName:   "inner_things",
			Contains:        SliceOf(InnerThingTypeMap),
		},
	},
}

var OuterPointerSliceThingTypeMap = StructMap{
	OuterPointerSliceThing{},
	[]MappedField{
		{
			StructFieldName: "InnerThings",
			JSONFieldName:   "inner_things",
			Contains:        SliceOf(InnerThingTypeMap),
		},
	},
}

var OuterPointerToSliceThingTypeMap = StructMap{
	OuterPointerToSliceThing{},
	[]MappedField{
		{
			StructFieldName: "InnerThings",
			JSONFieldName:   "inner_things",
			Contains:        SliceOf(InnerThingTypeMap),
		},
	},
}

var OtherInnerThingTypeMap = StructMap{
	OtherInnerThing{},
	[]MappedField{
		{
			StructFieldName: "Bar",
			JSONFieldName:   "bar",
			Validator:       String(1, 155),
			Optional:        true,
		},
	},
}

var OuterVariableThingTypeMap = StructMap{
	OuterVariableThing{},
	[]MappedField{
		{
			StructFieldName: "InnerType",
			JSONFieldName:   "inner_type",
			Validator:       String(1, 255),
		},
		{
			StructFieldName: "InnerValue",
			JSONFieldName:   "inner_thing",
			Contains: VariableType("InnerType", map[string]TypeMap{
				"foo": InnerThingTypeMap,
				"bar": OtherInnerThingTypeMap,
			}),
		},
	},
}

var BrokenOuterVariableThingTypeMap = StructMap{
	OtherOuterVariableThing{},
	[]MappedField{
		{
			StructFieldName: "InnerType",
			JSONFieldName:   "inner_type",
			Validator:       String(1, 255),
		},
		{
			StructFieldName: "InnerValue",
			JSONFieldName:   "inner_thing",
			Contains: VariableType("InnerTypeo", map[string]TypeMap{
				"foo": InnerThingTypeMap,
				"bar": OtherInnerThingTypeMap,
			}),
		},
	},
}

var ReadOnlyThingTypeMap = StructMap{
	ReadOnlyThing{},
	[]MappedField{
		{
			StructFieldName: "PrimaryKey",
			JSONFieldName:   "primary_key",
			ReadOnly:        true,
		},
	},
}

var TypoedThingTypeMap = StructMap{
	TypoedThing{},
	[]MappedField{
		{
			StructFieldName: "Incorrect",
			JSONFieldName:   "correct",
			Validator:       Boolean(),
		},
	},
}

var BrokenThingTypeMap = StructMap{
	BrokenThing{},
	[]MappedField{
		{
			StructFieldName: "Invalid",
			JSONFieldName:   "invalid",
			Validator:       brokenValidator{},
		},
	},
}

var TemplatableThingTypeMap = StructMap{
	TemplatableThing{},
	[]MappedField{
		{
			StructFieldName: "SomeField",
			JSONFieldName:   "some_field",
			Contains:        StringRenderer("{{.Context.Foo}}:{{.Value}}"),
		},
	},
}

var InnerNonMarshalableThingTypeMap = StructMap{
	InnerNonMarshalableThing{},
	[]MappedField{
		{
			StructFieldName: "Oops",
			JSONFieldName:   "oops",
		},
	},
}

var OuterNonMarshalableThingTypeMap = StructMap{
	OuterNonMarshalableThing{},
	[]MappedField{
		{
			StructFieldName: "InnerThing",
			JSONFieldName:   "inner_thing",
			Contains:        InnerNonMarshalableThingTypeMap,
		},
	},
}

var ThingWithSliceOfPrimitivesTypeMap = StructMap{
	ThingWithSliceOfPrimitives{},
	[]MappedField{
		{
			StructFieldName: "Strings",
			JSONFieldName:   "strings",
			Contains:        SliceOf(PrimitiveMap(String(1, 16))),
		},
	},
}

var ThingWithMapOfInterfacesTypeMap = StructMap{
	ThingWithMapOfInterfaces{},
	[]MappedField{
		{
			StructFieldName: "Interfaces",
			JSONFieldName:   "interfaces",
			Contains:        MapOf(PrimitiveMap(Interface())),
		},
	},
}

var ThingWithTimeSchema = StructMap{
	ThingWithTime{},
	[]MappedField{
		{
			StructFieldName: "HappenedAt",
			JSONFieldName:   "happened_at",
			Contains:        Time(),
		},
	},
}

var ThingWithEnumerableInterfaceSchema = StructMap{
	ThingWithEnumerableInterface{},
	[]MappedField{
		{
			StructFieldName: "ThanksGo",
			JSONFieldName:   "thanks",
			Validator:       OneOf("foo", "bar"),
		},
	},
}

var TestTypeMapper = NewTypeMapper(
	InnerThingTypeMap,
	OuterThingTypeMap,
	OuterPointerThingTypeMap,
	OuterInterfaceThingTypeMap,
	OuterSliceThingTypeMap,
	OuterPointerSliceThingTypeMap,
	OuterPointerToSliceThingTypeMap,
	OuterVariableThingTypeMap,
	BrokenOuterVariableThingTypeMap,
	ReadOnlyThingTypeMap,
	TypoedThingTypeMap,
	BrokenThingTypeMap,
	TemplatableThingTypeMap,
	InnerNonMarshalableThingTypeMap,
	OuterNonMarshalableThingTypeMap,
	ThingWithSliceOfPrimitivesTypeMap,
	ThingWithMapOfInterfacesTypeMap,
	ThingWithTimeSchema,
	ThingWithEnumerableInterfaceSchema,
)

func TestValidateInnerThing(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"foo": "fooz", "an_int": 10, "a_bool": true}`), v)
	if err != nil {
		t.Fatal(err)
	}
	if v.Foo != "fooz" {
		t.Fatal("Field Foo does not have expected value 'fooz':", v.Foo)
	}
}

func TestValidateOuterThing(t *testing.T) {
	v := &OuterThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"inner_thing": {"foo": "fooz"}}`), v)
	if err != nil {
		t.Fatal(err)
	}
	if v.InnerThing.Foo != "fooz" {
		t.Fatal("Inner field Foo does not have expected value 'fooz':", v.InnerThing.Foo)
	}
}

func TestValidateOuterSliceThing(t *testing.T) {
	v := &OuterSliceThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"inner_things": [{"foo": "fooz"}]}`), v)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.InnerThings) != 1 {
		t.Fatal("InnerThings should contain 1 element, instead contains", len(v.InnerThings))
	}
	if v.InnerThings[0].Foo != "fooz" {
		t.Fatal("InnerThing field Foo does not have expected value 'fooz':", v.InnerThings[0].Foo)
	}
}

func TestValidateOuterSliceThingInvalidElement(t *testing.T) {
	v := &OuterSliceThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"inner_things": [{"foo": "fooziswaytoolooong"}]}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'inner_things': index 0: 'foo': too long, may not be more than 12 characters" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateOuterSliceThingNotAList(t *testing.T) {
	v := &OuterSliceThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"inner_things": "foo"}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'inner_things': expected a list" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateReadOnlyThing(t *testing.T) {
	v := &ReadOnlyThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"primary_key": "foo"}`), v)
	if err != nil {
		t.Fatal(err)
	}
	if v.PrimaryKey != "" {
		t.Fatal("ReadOnly field unexpectedly set")
	}
}

func TestValidateReadOnlyThingValueNotProvided(t *testing.T) {
	v := &ReadOnlyThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{}`), v)
	if err != nil {
		t.Fatal(err)
	}
	if v.PrimaryKey != "" {
		t.Fatal("ReadOnly field unexpectedly set")
	}
}

func TestValidateUnregisteredThing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("No panic")
		}
	}()
	v := &UnregisteredThing{}
	TestTypeMapper.Unmarshal(EmptyContext, []byte(`{}`), v)
	t.Fatal("Unexpected success")
}

func TestValidateStringTypeMismatch(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"foo": 12.0}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'foo': not a string" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateStringTooShort(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"foo": ""}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'foo': too short, must be at least 1 characters" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateStringTooLong(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"foo": "thisvalueistoolong"}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'foo': too long, may not be more than 12 characters" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateBooleanTypeMismatch(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"a_bool": 12.0}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'a_bool': not a boolean" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateIntegerTypeMismatch(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"an_int": false}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'an_int': not an integer" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateIntegerNumericTypeMismatch(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"an_int": 12.1}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'an_int': not an integer" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateIntegerTooSmall(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"an_int": -1}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'an_int': too small, must be at least 0" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateIntegerTooLarge(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"an_int": 2048}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'an_int': too large, may not be larger than 10" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateWithUnexpectedError(t *testing.T) {
	v := &BrokenThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"invalid": "definitely"}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if _, ok := err.(*ValidationError); ok {
		t.Fatal("Unexpectedly received a proper ValidationError")
	}
	if err.Error() != "this should be a ValidationError" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestUnmarshalVariableTypeThing(t *testing.T) {
	{
		v := &OuterVariableThing{}
		err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"inner_type":"foo","inner_thing":{"foo":"bar"}}`), v)
		if err != nil {
			t.Fatal(err)
		}
		if v.InnerType != "foo" {
			t.Fatal("Unexpected value of InnerType:", v.InnerType)
		}
		it, ok := v.InnerValue.(*InnerThing)
		if !ok {
			t.Fatal("InnerValue has the wrong type:", reflect.TypeOf(v.InnerValue).String())
		}
		if it.Foo != "bar" {
			t.Fatal("Unexpected value of InnerThing.Foo:", it.Foo)
		}
	}
	{
		v := &OuterVariableThing{}
		err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"inner_type":"bar","inner_thing":{"bar":"foo"}}`), v)
		if err != nil {
			t.Fatal(err)
		}
		if v.InnerType != "bar" {
			t.Fatal("Unexpected value of InnerType:", v.InnerType)
		}
		it, ok := v.InnerValue.(*OtherInnerThing)
		if !ok {
			t.Fatal("InnerValue has the wrong type:", reflect.TypeOf(v.InnerValue).String())
		}
		if it.Bar != "foo" {
			t.Fatal("Unexpected value of InnerThing.Foo:", it.Bar)
		}
	}
}

func TestUnmarshalList(t *testing.T) {
	v := &InnerThing{}
	err := InnerThingTypeMap.Unmarshal(EmptyContext, nil, []interface{}{}, reflect.ValueOf(v))
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: expected an object" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestUnmarshalMissingRequiredField(t *testing.T) {
	v := &OuterThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: missing required field: inner_thing" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestUnmarshalNonPointer(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("No panic")
		}
		if r != "cannot unmarshal to non-pointer" {
			t.Fatal("Incorrect panic message", r)
		}
	}()
	v := InnerThing{}
	TestTypeMapper.Unmarshal(EmptyContext, []byte(`{}`), v)
}

func TestMarshalInnerThing(t *testing.T) {
	v := &InnerThing{
		Foo:   "bar",
		AnInt: 7,
		ABool: true,
	}
	data, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"foo":"bar","an_int":7,"a_bool":true}` {
		t.Fatal("Unexpected Marshal output:", string(data))
	}
}

func TestMarshalOuterThing(t *testing.T) {
	v := &OuterThing{
		InnerThing: InnerThing{
			Foo:   "bar",
			AnInt: 3,
			ABool: false,
		},
	}
	data, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"inner_thing":{"foo":"bar","an_int":3,"a_bool":false}}` {
		t.Fatal("Unexpected Marshal output:", string(data))
	}
}

func TestMarshalOuterPointerThing(t *testing.T) {
	v := &OuterPointerThing{
		InnerThing: &InnerThing{
			Foo:   "bar",
			AnInt: 3,
			ABool: false,
		},
	}
	data, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"inner_thing":{"foo":"bar","an_int":3,"a_bool":false}}` {
		t.Fatal("Unexpected Marshal output:", string(data))
	}
}

func TestUnmarshalOuterPointerThingWithNull(t *testing.T) {
	v := &OuterPointerThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"inner_thing": null}`), v)
	if err != nil {
		t.Fatal(err)
	}
	if v.InnerThing != nil {
		t.Fatal("Expected InnerThing to be nil")
	}
}

func TestMarshalOuterInterfaceThing(t *testing.T) {
	v := &OuterInterfaceThing{
		InnerThing: &InnerThing{
			Foo:   "bar",
			AnInt: 3,
			ABool: false,
		},
	}
	data, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"inner_thing":{"foo":"bar","an_int":3,"a_bool":false}}` {
		t.Fatal("Unexpected Marshal output:", string(data))
	}
}

func TestUnmarshalOuterInterfaceThing(t *testing.T) {
	v := &OuterInterfaceThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"inner_thing": {"foo":"bar","an_int":3,"a_bool":false}}`), v)
	if err != nil {
		t.Fatal(err)
	}

	innerThing, ok := v.InnerThing.(*InnerThing)
	if !ok {
		t.Fatal("InnerThing has an unexpected type")
	}

	if innerThing.Foo != "bar" {
		t.Fatal("InnerThing.Bar has an unexpected value")
	}

	if innerThing.AnInt != 3 {
		t.Fatal("InnerThing.AnInt has an unexpected value")
	}

	if innerThing.ABool != false {
		t.Fatal("InnerThing.ABool has an unexpected value")
	}
}

func TestUnmarshalOuterInterfaceThingWithNull(t *testing.T) {
	v := &OuterInterfaceThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"inner_thing": null}`), v)
	if err != nil {
		t.Fatal(err)
	}
	if v.InnerThing != nil {
		t.Fatal("Expected InnerThing to be nil")
	}
}

func TestMarshalOuterSliceThing(t *testing.T) {
	v := &OuterSliceThing{
		InnerThings: []InnerThing{
			{
				Foo:   "bar",
				AnInt: 3,
				ABool: false,
			},
		},
	}
	data, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"inner_things":[{"foo":"bar","an_int":3,"a_bool":false}]}` {
		t.Fatal("Unexpected Marshal output:", string(data))
	}

}

func TestMarshalOuterPointerSliceThing(t *testing.T) {
	v := &OuterPointerSliceThing{
		InnerThings: []*InnerThing{
			{
				Foo:   "bar",
				AnInt: 3,
				ABool: false,
			},
		},
	}
	data, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"inner_things":[{"foo":"bar","an_int":3,"a_bool":false}]}` {
		t.Fatal("Unexpected Marshal output:", string(data))
	}
}

func TestMarshalOuterPointerToSliceThing(t *testing.T) {
	v := &OuterPointerToSliceThing{
		InnerThings: &[]InnerThing{
			{
				Foo:   "bar",
				AnInt: 3,
				ABool: false,
			},
		},
	}
	data, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"inner_things":[{"foo":"bar","an_int":3,"a_bool":false}]}` {
		t.Fatal("Unexpected Marshal output:", string(data))
	}
}

func TestMarshalVariableTypeThing(t *testing.T) {
	{
		v := &OuterVariableThing{
			InnerType: "foo",
			InnerValue: &InnerThing{
				Foo: "test",
			},
		}

		data, err := TestTypeMapper.Marshal(EmptyContext, v)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != `{"inner_type":"foo","inner_thing":{"foo":"test","an_int":0,"a_bool":false}}` {
			t.Fatal("Unexpected Marshal output:", string(data))
		}
	}
	{
		v := &OuterVariableThing{
			InnerType: "bar",
			InnerValue: &OtherInnerThing{
				Bar: "test",
			},
		}

		data, err := TestTypeMapper.Marshal(EmptyContext, v)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != `{"inner_type":"bar","inner_thing":{"bar":"test"}}` {
			t.Fatal("Unexpected Marshal output:", string(data))
		}
	}
}

func TestMarshalBrokenVariableTypeThing(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("No panic")
		}
		if r != "no such underlying field: InnerTypeo" {
			t.Fatal("Incorrect panic message", r)
		}
	}()

	v := &OtherOuterVariableThing{
		InnerType: "foo",
		InnerValue: &InnerThing{
			Foo: "test",
		},
	}

	TestTypeMapper.Marshal(EmptyContext, v)
}

func TestMarshalVariableTypeThingInvalidTypeIdentifier(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("No panic")
		}
		if r != "variable type serialization error: validation error: invalid type identifier: 'wrong'" {
			t.Fatal("Incorrect panic message", r)
		}
	}()

	v := &OuterVariableThing{
		InnerType: "wrong",
		InnerValue: &InnerThing{
			Foo: "test",
		},
	}

	TestTypeMapper.Marshal(EmptyContext, v)
}

func TestMarshalNoSuchStructField(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("No panic")
		}
		if r != "no such underlying field: Incorrect" {
			t.Fatal("Incorrect panic message", r)
		}
	}()
	v := &TypoedThing{
		Correct: false,
	}
	TestTypeMapper.Marshal(EmptyContext, v)
}

func TestUnmarshalNoSuchStructField(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("No panic")
		}
		if r != "no such underlying field: Incorrect" {
			t.Fatal("Incorrect panic message", r)
		}
	}()
	v := &TypoedThing{}
	TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"correct": false}`), v)
}

func TestUnmarshalInvalidJSON(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"this is": "definitely invalid JSON]`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: unexpected end of JSON input" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestMarshalNonMarshalableThing(t *testing.T) {
	v := &OuterNonMarshalableThing{}
	_, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "json: error calling MarshalJSON for type jsonmap.NonMarshalableType: oops" {
		t.Fatal(err.Error())
	}
}

func TestMarshalSliceOfNonMarshalableThing(t *testing.T) {
	v := []OuterNonMarshalableThing{
		{},
	}
	_, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "json: error calling MarshalJSON for type jsonmap.NonMarshalableType: oops" {
		t.Fatal(err.Error())
	}
}

func TestMarshalIndent(t *testing.T) {
	v := &OuterThing{
		InnerThing: InnerThing{
			Foo:   "bar",
			AnInt: 3,
			ABool: false,
		},
	}
	expected := "{\n" +
		"    \"inner_thing\": {\n" +
		"        \"foo\": \"bar\",\n" +
		"        \"an_int\": 3,\n" +
		"        \"a_bool\": false\n" +
		"    }\n" +
		"}"
	data, err := TestTypeMapper.MarshalIndent(EmptyContext, v, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Fatal("Unexpected Marshal output:", string(data), expected)
	}
}

func TestMarshalSlice(t *testing.T) {
	v := []InnerThing{
		{
			Foo:   "bar",
			AnInt: 3,
			ABool: false,
		},
		{
			Foo:   "bam",
			AnInt: 4,
			ABool: true,
		},
	}
	expected := `[{"foo":"bar","an_int":3,"a_bool":false},{"foo":"bam","an_int":4,"a_bool":true}]`
	data, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Fatal("Unexpected Marshal output:", string(data), expected)
	}
}

func TestMarshalSliceOfPointers(t *testing.T) {
	v := []*InnerThing{
		&InnerThing{
			Foo:   "bar",
			AnInt: 3,
			ABool: false,
		},
		&InnerThing{
			Foo:   "bam",
			AnInt: 4,
			ABool: true,
		},
	}
	expected := `[{"foo":"bar","an_int":3,"a_bool":false},{"foo":"bam","an_int":4,"a_bool":true}]`
	data, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Fatal("Unexpected Marshal output:", string(data), expected)
	}
}

func TestMarshalTemplatableThing(t *testing.T) {
	ctx := struct {
		Foo string
	}{
		Foo: "foo",
	}

	v := &TemplatableThing{
		SomeField: "bar",
	}

	expected := `{"some_field":"foo:bar"}`
	data, err := TestTypeMapper.Marshal(ctx, v)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != expected {
		t.Fatal("Unexpected Marshal output:", string(data), expected)
	}
}

func TestMarshalThingWithSliceOfPrimitives(t *testing.T) {
	v := ThingWithSliceOfPrimitives{
		Strings: []string{"foo", "bar"},
	}

	expected := `{"strings":["foo","bar"]}`
	data, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Fatal("Unexpected Marshal output:", string(data), expected)
	}
}

func TestValidateThingWithSliceOfPrimitives(t *testing.T) {
	original := `{"strings":["foo","bar"]}`
	v := &ThingWithSliceOfPrimitives{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(original), v)
	if err != nil {
		t.Fatal(err)
	}

	data, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatal("Unoriginal Marshal output:", string(data), original)
	}
}

func TestMarshalThingWithMapOfInterfaces(t *testing.T) {
	interfaces := map[string]interface{}{
		"foo": "bar",
		"baz": 10,
		"qux": []string{"dang"},
	}

	v := ThingWithMapOfInterfaces{
		Interfaces: interfaces,
	}

	data, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err != nil {
		t.Fatal(err)
	}

	expected, err := json.Marshal(map[string]interface{}{"interfaces": interfaces})
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != string(expected) {
		t.Fatal("unexpected Marshal output", string(data), string(expected))
	}
}

func TestValidateThingWithMapOfInterfaces(t *testing.T) {
	original := `{"interfaces":{"baz":10,"dux":null,"foo":"bar","qux":["dang"]}}`
	v := &ThingWithMapOfInterfaces{}
	err := TestTypeMapper.Unmarshal(EmptyContext, []byte(original), v)
	if err != nil {
		t.Fatal(err)
	}

	data, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatal("Unoriginal Marshal output:", string(data), original)
	}
}

func TestMarshalThingWithTime(t *testing.T) {
	ts, err := time.Parse(time.RFC822, time.RFC822)
	if err != nil {
		panic(err)
	}

	v := ThingWithTime{
		HappenedAt: ts,
	}

	expected := `{"happened_at":"2006-01-02T15:04:00Z"}`
	data, err := TestTypeMapper.Marshal(EmptyContext, v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Fatal("Unexpected Marshal output:", string(data), expected)
	}
}

func TestUnmarshalThingWithTime(t *testing.T) {
	ts, err := time.Parse(time.RFC822, time.RFC822)
	if err != nil {
		panic(err)
	}

	v := &ThingWithTime{}

	err = TestTypeMapper.Unmarshal(EmptyContext, []byte(`{"happened_at":"2006-01-02T15:04:00Z"}`), v)
	if err != nil {
		t.Fatal(err)
	}

	if !ts.Equal(v.HappenedAt) {
		t.Fatal("Timestamp mismatch:", v.HappenedAt, ts)
	}
}

func TestGenericUnmarshalInvalidInput(t *testing.T) {
	invalidCases := []struct {
		Input        string
		Into         ThingWithEnumerableInterface
		ErrorMessage string
	}{
		{
			Input:        `{"thanks": "baz"}`,
			Into:         ThingWithEnumerableInterface{},
			ErrorMessage: `validation error: 'thanks': Value must be one of: ["foo","bar"]`,
		},
		{
			Input:        `{"thanks": 12}`,
			Into:         ThingWithEnumerableInterface{},
			ErrorMessage: `validation error: 'thanks': not a string`,
		},
	}

	for _, invalidCase := range invalidCases {
		dest := invalidCase.Into
		err := TestTypeMapper.Unmarshal(EmptyContext, []byte(invalidCase.Input), &dest)
		require.Error(t, err)
		require.Equal(t, invalidCase.ErrorMessage, err.Error())
	}
}

func TestValidThingWithEnumerableInterface(t *testing.T) {
	validCases := []struct {
		Input    string
		Expected ThingWithEnumerableInterface
	}{
		{
			Input: `{"thanks": "foo"}`,
			Expected: ThingWithEnumerableInterface{
				ThanksGo: "foo",
			},
		},
		{
			Input: `{"thanks": "bar"}`,
			Expected: ThingWithEnumerableInterface{
				ThanksGo: "bar",
			},
		},
	}

	for _, validCase := range validCases {
		dest := validCase.Expected
		err := TestTypeMapper.Unmarshal(EmptyContext, []byte(validCase.Input), &dest)
		require.Nil(t, err)
		require.EqualValues(t, validCase.Expected, dest)
	}
}
