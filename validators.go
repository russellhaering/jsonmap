package jsonmap

type stringValidator struct {
	minLen int
	maxLen int
}

func (v *stringValidator) Validate(value interface{}) (interface{}, error) {
	s, ok := value.(string)
	if !ok {
		return nil, NewValidationError("not a string")
	}

	if len(s) < v.minLen {
		return nil, NewValidationError("too short, must be at least %d characters", v.minLen)
	}

	if len(s) > v.maxLen {
		return nil, NewValidationError("too long, may not be more than %d characters", v.maxLen)
	}

	return s, nil
}

func String(minLen int, maxLen int) Validator {
	return &stringValidator{
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
	return int(f), nil
}

func Integer(minVal, maxVal int) Validator {
	return &integerValidator{
		minVal: minVal,
		maxVal: maxVal,
	}
}
