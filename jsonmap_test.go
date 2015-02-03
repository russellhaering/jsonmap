package jsonmap

import "testing"

type InnerThing struct {
	Foo   string
	AnInt int
	ABool bool
}

type OuterThing struct {
	Bar        string
	InnerThing InnerThing
}

type UnregisteredThing struct {
}

var InnerThingTypeMap = TypeMap{
	InnerThing{},
	[]MappedField{
		{
			StructFieldName: "Foo",
			JSONFieldName:   "foo",
			Validator:       String(1, 255),
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
			StructFieldName: "Bar",
			JSONFieldName:   "bar",
			Validator:       String(1, 255),
		},
		{
			StructFieldName: "InnerThing",
			JSONFieldName:   "inner_thing",
			Contains:        InnerThingTypeMap,
		},
	},
}

var TestTypeMapper = NewTypeMapper(
	InnerThingTypeMap,
	OuterThingTypeMap,
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
	err := TestTypeMapper.Unmarshal([]byte(`{"bar": "bazam", "inner_thing": {"foo": "fooz"}}`), v)
	if err != nil {
		t.Fatal(err)
	}
	if v.InnerThing.Foo != "fooz" {
		t.Fatal("Inner field Foo does not have expected value 'fooz':", v.InnerThing.Foo)
	}
	if v.Bar != "bazam" {
		t.Fatal("Outer field Bar does not have expected value 'bazam':", v.Bar)
	}
}
