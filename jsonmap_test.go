package jsonmap

import (
	"errors"
	"reflect"
	"testing"
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

type OuterSliceThing struct {
	InnerThings []InnerThing
}

type OuterPointerSliceThing struct {
	InnerThings []*InnerThing
}

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

type NonMarshalableThing struct{}

func (t NonMarshalableThing) MarshalJSON() ([]byte, error) {
	return nil, errors.New("oops")
}

type OuterNonMarshalableThing struct {
	Oops NonMarshalableThing
}

var InnerThingTypeMap = TypeMap{
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

var OuterThingTypeMap = TypeMap{
	OuterThing{},
	[]MappedField{
		{
			StructFieldName: "InnerThing",
			JSONFieldName:   "inner_thing",
			Contains:        InnerThingTypeMap,
		},
	},
}

var OuterPointerThingTypeMap = TypeMap{
	OuterPointerThing{},
	[]MappedField{
		{
			StructFieldName: "InnerThing",
			JSONFieldName:   "inner_thing",
			Contains:        InnerThingTypeMap,
		},
	},
}

var OuterSliceThingTypeMap = TypeMap{
	OuterSliceThing{},
	[]MappedField{
		{
			StructFieldName: "InnerThings",
			JSONFieldName:   "inner_things",
			Contains:        SliceOf(InnerThingTypeMap),
		},
	},
}

var OuterPointerSliceThingTypeMap = TypeMap{
	OuterPointerSliceThing{},
	[]MappedField{
		{
			StructFieldName: "InnerThings",
			JSONFieldName:   "inner_things",
			Contains:        SliceOf(InnerThingTypeMap),
		},
	},
}

var ReadOnlyThingTypeMap = TypeMap{
	ReadOnlyThing{},
	[]MappedField{
		{
			StructFieldName: "PrimaryKey",
			JSONFieldName:   "primary_key",
			ReadOnly:        true,
		},
	},
}

var TypoedThingTypeMap = TypeMap{
	TypoedThing{},
	[]MappedField{
		{
			StructFieldName: "Incorrect",
			JSONFieldName:   "correct",
			Validator:       Boolean(),
		},
	},
}

var BrokenThingTypeMap = TypeMap{
	BrokenThing{},
	[]MappedField{
		{
			StructFieldName: "Invalid",
			JSONFieldName:   "invalid",
			Validator:       brokenValidator{},
		},
	},
}

var OuterNonMarshalableThingTypeMap = TypeMap{
	OuterNonMarshalableThing{},
	[]MappedField{
		{
			StructFieldName: "Oops",
			JSONFieldName:   "oops",
		},
	},
}

var TestTypeMapper = NewTypeMapper(
	InnerThingTypeMap,
	OuterThingTypeMap,
	OuterPointerThingTypeMap,
	OuterSliceThingTypeMap,
	OuterPointerSliceThingTypeMap,
	ReadOnlyThingTypeMap,
	TypoedThingTypeMap,
	BrokenThingTypeMap,
	OuterNonMarshalableThingTypeMap,
)

func TestValidateInnerThing(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"foo": "fooz", "an_int": 10, "a_bool": true}`), v)
	if err != nil {
		t.Fatal(err)
	}
	if v.Foo != "fooz" {
		t.Fatal("Field Foo does not have expected value 'fooz':", v.Foo)
	}
}

func TestValidateOuterThing(t *testing.T) {
	v := &OuterThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"inner_thing": {"foo": "fooz"}}`), v)
	if err != nil {
		t.Fatal(err)
	}
	if v.InnerThing.Foo != "fooz" {
		t.Fatal("Inner field Foo does not have expected value 'fooz':", v.InnerThing.Foo)
	}
}

func TestValidateOuterSliceThing(t *testing.T) {
	v := &OuterSliceThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"inner_things": [{"foo": "fooz"}]}`), v)
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
	err := TestTypeMapper.Unmarshal([]byte(`{"inner_things": [{"foo": "fooziswaytoolooong"}]}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'inner_things': index 0: 'foo': too long, may not be more than 12 characters" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateOuterSliceThingNotAList(t *testing.T) {
	v := &OuterSliceThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"inner_things": "foo"}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'inner_things': expected a list" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateReadOnlyThing(t *testing.T) {
	v := &ReadOnlyThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"primary_key": "foo"}`), v)
	if err != nil {
		t.Fatal(err)
	}
	if v.PrimaryKey != "" {
		t.Fatal("ReadOnly field unexpectedly set")
	}
}

func TestValidateReadOnlyThingValueNotProvided(t *testing.T) {
	v := &ReadOnlyThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{}`), v)
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
	TestTypeMapper.Unmarshal([]byte(`{}`), v)
	t.Fatal("Unexpected success")
}

func TestValidateStringTypeMismatch(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"foo": 12.0}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'foo': not a string" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateStringTooShort(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"foo": ""}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'foo': too short, must be at least 1 characters" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateStringTooLong(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"foo": "thisvalueistoolong"}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'foo': too long, may not be more than 12 characters" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateBooleanTypeMismatch(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"a_bool": 12.0}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'a_bool': not a boolean" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateIntegerTypeMismatch(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"an_int": false}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'an_int': not an integer" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateIntegerNumericTypeMismatch(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"an_int": 12.1}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'an_int': not an integer" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateIntegerTooSmall(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"an_int": -1}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'an_int': too small, must be at least 0" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateIntegerTooLarge(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"an_int": 2048}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: 'an_int': too large, may not be larger than 10" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateWithUnexpectedError(t *testing.T) {
	v := &BrokenThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"invalid": "definitely"}`), v)
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

func TestUnmarshalList(t *testing.T) {
	v := &InnerThing{}
	err := InnerThingTypeMap.Unmarshal([]interface{}{}, reflect.ValueOf(v))
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: expected an object" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestUnmarshalMissingRequiredField(t *testing.T) {
	v := &OuterThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{}`), v)
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
	TestTypeMapper.Unmarshal([]byte(`{}`), v)
}

func TestMarshalInnerThing(t *testing.T) {
	v := &InnerThing{
		Foo:   "bar",
		AnInt: 7,
		ABool: true,
	}
	data, err := TestTypeMapper.Marshal(v)
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
	data, err := TestTypeMapper.Marshal(v)
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
	data, err := TestTypeMapper.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"inner_thing":{"foo":"bar","an_int":3,"a_bool":false}}` {
		t.Fatal("Unexpected Marshal output:", string(data))
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
	data, err := TestTypeMapper.Marshal(v)
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
	data, err := TestTypeMapper.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"inner_things":[{"foo":"bar","an_int":3,"a_bool":false}]}` {
		t.Fatal("Unexpected Marshal output:", string(data))
	}
}

func TestMarshalNoSuchStructField(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("No panic")
		}
		if r != "No such underlying field: Incorrect" {
			t.Fatal("Incorrect panic message", r)
		}
	}()
	v := &TypoedThing{
		Correct: false,
	}
	TestTypeMapper.Marshal(v)
}

func TestUnmarshalNoSuchStructField(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("No panic")
		}
		if r != "No such underlying field: Incorrect" {
			t.Fatal("Incorrect panic message", r)
		}
	}()
	v := &TypoedThing{}
	TestTypeMapper.Unmarshal([]byte(`{"correct": false}`), v)
}

func TestUnmarshalInvalidJSON(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"this is": "definitely invalid JSON]`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "validation error: unexpected end of JSON input" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestMarshalNonMarshalableThing(t *testing.T) {
	v := &OuterNonMarshalableThing{}
	_, err := TestTypeMapper.Marshal(v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "json: error calling MarshalJSON for type jsonmap.NonMarshalableThing: oops" {
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
	data, err := TestTypeMapper.MarshalIndent(v, "", "    ")
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
	data, err := TestTypeMapper.Marshal(v)
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
	data, err := TestTypeMapper.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Fatal("Unexpected Marshal output:", string(data), expected)
	}
}
