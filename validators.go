package jsonmap

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
)

var uuidRegex = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type StringValidator struct {
	minLen int
	maxLen int
	re     *regexp.Regexp
}

func (v *StringValidator) ValidateString(s string) (string, error) {
	if len(s) < v.minLen {
		return "", NewValidationError("too short, must be at least %d characters", v.minLen)
	}

	if len(s) > v.maxLen {
		return "", NewValidationError("too long, may not be more than %d characters", v.maxLen)
	}

	if v.re != nil && !v.re.MatchString(s) {
		return "", NewValidationError("must match regular expression: %s", v.re.String())
	}
	return s, nil
}

func (v *StringValidator) Validate(value interface{}) (interface{}, error) {
	s, ok := value.(string)
	if !ok {
		return nil, NewValidationError("not a string")
	}

	return v.ValidateString(s)
}

func (v *StringValidator) Regex(re *regexp.Regexp) *StringValidator {
	v.re = re
	return v
}

func String(minLen int, maxLen int) *StringValidator {
	return &StringValidator{
		minLen: minLen,
		maxLen: maxLen,
	}
}

type booleanValidator struct{}

func (v *booleanValidator) Validate(value interface{}) (interface{}, error) {
	b, ok := value.(bool)
	if !ok {
		return nil, NewValidationError("not a boolean")
	}
	return b, nil
}

func Boolean() Validator {
	return &booleanValidator{}
}

// TODO: The spectrum of numeric types deserves more thought. Do we ship
// independent validators for each?
type integerValidator struct {
	minVal int
	maxVal int
}

func (v *integerValidator) Validate(value interface{}) (interface{}, error) {
	// Numeric values come in as a float64. This almost certainly has some weird
	// properties in extreme cases, but JSON probably isn't the right choice in
	// those cases.
	f, ok := value.(float64)
	if !ok || float64(int(f)) != f {
		return nil, NewValidationError("not an integer")
	}

	i := int(f)
	if i < v.minVal {
		return nil, NewValidationError("too small, must be at least %d", v.minVal)
	}

	if i > v.maxVal {
		return nil, NewValidationError("too large, may not be larger than %d", v.maxVal)
	}

	return i, nil
}

func Integer(minVal, maxVal int) Validator {
	return &integerValidator{
		minVal: minVal,
		maxVal: maxVal,
	}
}

type interfaceValidator struct{}

func (v *interfaceValidator) Validate(value interface{}) (interface{}, error) {
	return value, nil
}

func Interface() *interfaceValidator {
	return &interfaceValidator{}
}

type LossyUint64Validator struct {
	minVal uint64
	maxVal uint64
}

func (v *LossyUint64Validator) Validate(value interface{}) (interface{}, error) {
	f, ok := value.(float64)
	if !ok || float64(uint64(f)) != f {
		return nil, NewValidationError("not an integer")
	}

	i := uint64(f)
	if i < v.minVal {
		return nil, NewValidationError("too small, must be at least %d", v.minVal)
	}

	if i > v.maxVal {
		return nil, NewValidationError("too large, may not be larger than %d", v.maxVal)
	}

	return i, nil
}

func (v *LossyUint64Validator) Min(min uint64) {
	v.minVal = min
}

func (v *LossyUint64Validator) Max(max uint64) {
	v.maxVal = max
}

// Validate numbers as a uint64. In this process they will be stored as a
// float64, which can lead to a loss of precision as high as 1024(?).
func LossyUint64() *LossyUint64Validator {
	return &LossyUint64Validator{
		minVal: 0,
		maxVal: math.MaxUint64,
	}
}

type UUIDStringValidator struct{}

func (v *UUIDStringValidator) Validate(value interface{}) (interface{}, error) {
	s, ok := value.(string)
	if !ok {
		return nil, NewValidationError("not a string")
	}

	return v.ValidateString(s)
}

func (v *UUIDStringValidator) ValidateString(value string) (string, error) {
	if !uuidRegex.MatchString(value) {
		return "", NewValidationError("not a valid UUID")
	}

	return value, nil
}

func UUIDString() *UUIDStringValidator {
	return &UUIDStringValidator{}
}

type ssm struct {
	sv *StringValidator
}

// Used for StringsArray, which has a "V" field containing []string.
// Optionally can take a string validator to apply to each entry.
func StringsSliceMapper(sv *StringValidator) TypeMap {
	return &ssm{sv: sv}
}

func (ss *ssm) Unmarshal(ctx Context, parent *reflect.Value, partial interface{}, dstValue reflect.Value) error {
	var err error
	v := dstValue.FieldByName("V")

	underlying := v.Interface()
	if _, ok := underlying.([]string); !ok {
		panic("target field V for StringsSliceMapper is not a []string")
	}

	if partial == nil {
		v.Set(reflect.ValueOf([]string{}))
		return nil
	}

	data, ok := partial.([]interface{})
	if !ok {
		return NewValidationError("expected a list")
	}

	rv := make([]string, len(data))

	for i, dv := range data {
		s, ok := dv.(string)
		if !ok {
			return fmt.Errorf("Error converting %#v to string", dv)
		}

		if ss.sv != nil {
			s, err = ss.sv.ValidateString(s)
			if err != nil {
				return err
			}
		}

		rv[i] = s
	}

	v.Set(reflect.ValueOf(rv))

	return nil
}

func (s *ssm) Marshal(ctx Context, parent *reflect.Value, src reflect.Value) (json.Marshaler, error) {
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}

	v := src.FieldByName("V")

	data, err := json.Marshal(v.Interface())
	if err != nil {
		return nil, err
	}

	return RawMessage{data}, nil
}
