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
	MinLen   int
	MaxLen   int
	RE       *regexp.Regexp
	REErrMsg string
}

func (v *StringValidator) ValidateString(s string) (string, error) {
	if len(s) < v.MinLen {
		return "", NewValidationError("too short, must be at least %d characters", v.MinLen)
	}

	if len(s) > v.MaxLen {
		return "", NewValidationError("too long, may not be more than %d characters", v.MaxLen)
	}

	if v.RE != nil && !v.RE.MatchString(s) {
		if v.REErrMsg != "" {
			return "", NewValidationError(v.REErrMsg)
		}

		return "", NewValidationError("must match regular expression: %s", v.RE.String())
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
	v.RE = re
	return v
}

func (v *StringValidator) RegexError(re *regexp.Regexp, errMsg string) *StringValidator {
	v.RE = re
	v.REErrMsg = errMsg
	return v
}

func String(minLen int, maxLen int) *StringValidator {
	return &StringValidator{
		MinLen: minLen,
		MaxLen: maxLen,
	}
}

type BooleanValidator struct{}

func (v *BooleanValidator) Validate(value interface{}) (interface{}, error) {
	b, ok := value.(bool)
	if !ok {
		return nil, NewValidationError("not a boolean")
	}
	return b, nil
}

func Boolean() Validator {
	return &BooleanValidator{}
}

// TODO: The spectrum of numeric types deserves more thought. Do we ship
// independent validators for each?
type IntegerValidator struct {
	MinVal int64
	MaxVal int64
}

func (v *IntegerValidator) Validate(value interface{}) (interface{}, error) {
	// Numeric values come in as a float64. This almost certainly has some weird
	// properties in extreme cases, but JSON probably isn't the right choice in
	// those cases.
	f, ok := value.(float64)
	if !ok || float64(int64(f)) != f {
		return nil, NewValidationError("not an integer")
	}

	i := int64(f)
	if i < v.MinVal {
		return nil, NewValidationError("too small, must be at least %d", v.MinVal)
	}

	if i > v.MaxVal {
		return nil, NewValidationError("too large, may not be larger than %d", v.MaxVal)
	}

	return i, nil
}

func Integer(minVal, maxVal int64) Validator {
	return &IntegerValidator{
		MinVal: minVal,
		MaxVal: maxVal,
	}
}

type InterfaceValidator struct{}

func (v *InterfaceValidator) Validate(value interface{}) (interface{}, error) {
	return value, nil
}

func Interface() *InterfaceValidator {
	return &InterfaceValidator{}
}

type LossyUint64Validator struct {
	MinVal uint64
	MaxVal uint64
}

func (v *LossyUint64Validator) Validate(value interface{}) (interface{}, error) {
	f, ok := value.(float64)
	if !ok || float64(uint64(f)) != f {
		return nil, NewValidationError("not an integer")
	}

	i := uint64(f)
	if i < v.MinVal {
		return nil, NewValidationError("too small, must be at least %d", v.MinVal)
	}

	if i > v.MaxVal {
		return nil, NewValidationError("too large, may not be larger than %d", v.MaxVal)
	}

	return i, nil
}

func (v *LossyUint64Validator) Min(min uint64) {
	v.MinVal = min
}

func (v *LossyUint64Validator) Max(max uint64) {
	v.MaxVal = max
}

// Validate numbers as a uint64. In this process they will be stored as a
// float64, which can lead to a loss of precision as high as 1024(?).
func LossyUint64() *LossyUint64Validator {
	return &LossyUint64Validator{
		MinVal: 0,
		MaxVal: math.MaxUint64,
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

type StringsSliceMapper struct {
	StringValidator *StringValidator
}

// Used for StringsArray, which has a "V" field containing []string.
// Optionally can take a string validator to apply to each entry.
func NewStringsSliceMapper(sv *StringValidator) TypeMap {
	return &StringsSliceMapper{StringValidator: sv}
}

func (ss *StringsSliceMapper) Unmarshal(ctx Context, parent *reflect.Value, partial interface{}, dstValue reflect.Value) error {
	var err error
	v := dstValue.FieldByName("V")

	underlying := v.Interface()
	if _, ok := underlying.([]string); !ok {
		panic("target field V for NewStringsSliceMapper is not a []string")
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

		if ss.StringValidator != nil {
			s, err = ss.StringValidator.ValidateString(s)
			if err != nil {
				return err
			}
		}

		rv[i] = s
	}

	v.Set(reflect.ValueOf(rv))

	return nil
}

func (s *StringsSliceMapper) Marshal(ctx Context, parent *reflect.Value, src reflect.Value) (json.Marshaler, error) {
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

type EnumeratedValuesValidator struct {
	AllowedSlice  []string
	AllowedValues map[string]struct{}
}

func (v *EnumeratedValuesValidator) Validate(value interface{}) (interface{}, error) {
	s, ok := value.(string)
	if !ok {
		return nil, NewValidationError("not a string")
	}
	_, ok = v.AllowedValues[s]

	if !ok {
		serialized, err := json.Marshal(v.AllowedSlice)
		if err != nil {
			// AllowedSlice should be a static value provided by the programmer,
			// so an error serializing it definitely represents a progrramming error.
			panic(err)
		}

		// If we want to use the invalid string value for error messages, return the string value instead of nil and in
		// the calling function, check if the return value is valid instead of checking if an error was returned, when
		// setting that value in the dest object (this valid check would handle if the input value is not a string)
		// return s, NewValidationError("Value must be one of: %s", string(serialized))
		return nil, NewValidationError("Value must be one of: %s", string(serialized))
	}

	return value, nil
}

func OneOf(allowed ...string) Validator {
	v := &EnumeratedValuesValidator{
		AllowedSlice:  allowed,
		AllowedValues: map[string]struct{}{},
	}

	for _, value := range allowed {
		v.AllowedValues[value] = struct{}{}
	}

	return v
}

func KeyFromVariableTypeMap(m map[string]TypeMap) Validator {
	keys := make([]string, 0, len(m))

	for key := range m {
		keys = append(keys, key)
	}

	return OneOf(keys...)
}
