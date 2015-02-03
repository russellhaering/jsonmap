[![Build Status](https://travis-ci.org/russellhaering/jsonmap.svg?branch=master)](https://travis-ci.org/russellhaering/jsonmap)

# JSON validator and mapper

Validate JSON, and deserialize it into a structure without modifying or tagging
the structure.

## Example

```go
package main

import "github.com/russellhaering/jsonmap"

type Dog struct {
	Name string
	Age  int
}

var DogTypeMap = jsonmap.TypeMap{
	Dog{},
	[]jsonmap.MappedField{
		{
			StructFieldName: "Name",
			JSONFieldName:   "name",
			Validator:       jsonmap.String(1, 128),
		},
		{
			StructFieldName: "Age",
			JSONFieldName:   "age",
			Validator:       jsonmap.Integer(0, 1024),
		},
	},
}

var DemoTypeMapper = jsonmap.NewTypeMapper(
	DogTypeMap,
)

func main() {
	d := &Dog{}
	err := DemoTypeMapper.Unmarshal([]byte(`{"name": "Spot", "age": 4}`), d)
	if err != nil {
		panic(err)
	}
	println(d.Name)
	println(d.Age)
}
```
## Why?

Use of struct tags to describe how to map JSON encourages bad design patterns.
Developers end putting JSON struct tags onto database objects (which should
have no idea how some other layer might choose to serialize them) or writing
tons of boiler plate to map API level structures to database objects. Using
`jsonmap` is a way to avoid that boilerplate.
