package jsonmap

import (
	"reflect"
	"testing"
)

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

type UnregisteredThing struct {
}

type TypoedThing struct {
	Correct bool
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

var TestTypeMapper = NewTypeMapper(
	InnerThingTypeMap,
	OuterThingTypeMap,
	OuterPointerThingTypeMap,
	TypoedThingTypeMap,
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
	if err.Error() != "error validating field 'foo': not a string" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateStringTooShort(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"foo": ""}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "error validating field 'foo': too short, must be at least 1 characters" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateStringTooLong(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"foo": "thisvalueistoolong"}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "error validating field 'foo': too long, may not be more than 12 characters" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateBooleanTypeMismatch(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"a_bool": 12.0}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "error validating field 'a_bool': not a boolean" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateIntegerTypeMismatch(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"an_int": false}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "error validating field 'an_int': not an integer" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateIntegerNumericTypeMismatch(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"an_int": 12.1}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "error validating field 'an_int': not an integer" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateIntegerTooSmall(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"an_int": -1}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "error validating field 'an_int': too small, must be at least 0" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestValidateIntegerTooLarge(t *testing.T) {
	v := &InnerThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{"an_int": 2048}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "error validating field 'an_int': too large, may not be larger than 10" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestUnmarshalList(t *testing.T) {
	v := &InnerThing{}
	err := InnerThingTypeMap.Unmarshal([]interface{}{}, reflect.ValueOf(v))
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "expected a JSON object" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestUnmarshalMissingRequiredField(t *testing.T) {
	v := &OuterThing{}
	err := TestTypeMapper.Unmarshal([]byte(`{}`), v)
	if err == nil {
		t.Fatal("Unexpected success")
	}
	if err.Error() != "missing required field: inner_thing" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

func TestUnmarshalNonPointer(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("No panic")
		}
		if r != "cannot map a non-pointer" {
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
	if string(data) != `{"a_bool":true,"an_int":7,"foo":"bar"}` {
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
	if string(data) != `{"inner_thing":{"a_bool":false,"an_int":3,"foo":"bar"}}` {
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
	if string(data) != `{"inner_thing":{"a_bool":false,"an_int":3,"foo":"bar"}}` {
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
		"        \"a_bool\": false,\n" +
		"        \"an_int\": 3,\n" +
		"        \"foo\": \"bar\"\n" +
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
