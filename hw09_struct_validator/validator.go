package hw09structvalidator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	ErrNotStruct       = errors.New("input is not a struct")
	ErrInvalidTag      = errors.New("invalid validate tag")
	ErrUnsupportedType = errors.New("unsupported field type")
)

var (
	ErrLen      = errors.New("invalid string length")
	ErrTooSmall = errors.New("value is less than allowed minimum")
	ErrTooBig   = errors.New("value is greater than allowed maximum")
	ErrNotInSet = errors.New("value is not in the allowed set")
	ErrRegexp   = errors.New("string does not match the required pattern")
)

type ValidationError struct {
	Field string
	Err   error
}

type ValidationErrors []ValidationError

type rule struct {
	name string
	arg  string
}

type elemValidator func(fv reflect.Value, name, tag string) (ValidationErrors, error)

func (v ValidationErrors) Error() string {
	msgs := make([]string, 0, len(v))
	for _, e := range v {
		msgs = append(msgs, e.Field+": "+e.Err.Error())
	}
	return strings.Join(msgs, "; ")
}

func Validate(v interface{}) error {
	if v == nil {
		return ErrNotStruct
	}

	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ErrNotStruct
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ErrNotStruct
	}

	errs, progErr := validateStruct(rv, "")
	if progErr != nil {
		return progErr
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateStruct(rv reflect.Value, prefix string) (ValidationErrors, error) {
	typ := rv.Type()
	var errs ValidationErrors

	for i := range typ.NumField() {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		tag, ok := field.Tag.Lookup("validate")
		if !ok || tag == "" {
			continue
		}

		name := prefix + field.Name

		fieldErrs, progErr := validateField(rv.Field(i), name, tag)
		if progErr != nil {
			return nil, progErr
		}
		errs = append(errs, fieldErrs...)
	}

	return errs, nil
}

func validateField(fv reflect.Value, name, tag string) (ValidationErrors, error) {
	switch fv.Kind() { //nolint:exhaustive
	case reflect.Struct:
		if tag != "nested" {
			return nil, fmt.Errorf("field %s: %w", name, ErrUnsupportedType)
		}
		return validateStruct(fv, name+".")
	case reflect.String:
		return validateString(fv, name, tag)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return validateInt(fv, name, tag)
	case reflect.Slice:
		return validateSlice(fv, name, tag)
	default:
		return nil, fmt.Errorf("field %s: %w", name, ErrUnsupportedType)
	}
}

func validateSlice(fv reflect.Value, name, tag string) (ValidationErrors, error) {
	var validator elemValidator
	switch fv.Type().Elem().Kind() { //nolint:exhaustive
	case reflect.String:
		validator = validateString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		validator = validateInt
	default:
		return nil, fmt.Errorf("field %s: %w", name, ErrUnsupportedType)
	}

	var errs ValidationErrors
	for i := 0; i < fv.Len(); i++ {
		field := fv.Index(i)
		validationErrs, progErr := validator(field, name, tag)
		if progErr != nil {
			return nil, progErr
		}

		errs = append(errs, validationErrs...)
	}

	return errs, nil
}

func parseRules(tag string) ([]rule, error) {
	tokens := strings.Split(tag, "|")
	rules := make([]rule, 0, len(tokens))

	for _, token := range tokens {
		name, arg, ok := strings.Cut(token, ":")
		if !ok || name == "" || arg == "" {
			return nil, ErrInvalidTag
		}

		rules = append(rules, rule{name, arg})
	}

	return rules, nil
}

func validateString(fv reflect.Value, name, tag string) (ValidationErrors, error) {
	rules, err := parseRules(tag)
	if err != nil {
		return nil, fmt.Errorf("field %s: %w", name, err)
	}

	value := fv.String()
	length := utf8.RuneCountInString(value)
	var errs ValidationErrors

	for _, r := range rules {
		switch r.name {
		case "len":
			want, err := strconv.Atoi(r.arg)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w: %q is not a number", name, ErrInvalidTag, r.arg)
			}
			if length != want {
				errs = append(errs, ValidationError{
					Field: name,
					Err:   fmt.Errorf("%w: %q has length %d, expected %d", ErrLen, value, length, want),
				})
			}
		case "regexp":
			re, err := regexp.Compile("^(?:" + r.arg + ")$")
			if err != nil {
				return nil, fmt.Errorf("field %s: %w: %q is not a valid regexp", name, ErrInvalidTag, r.arg)
			}
			if !re.MatchString(value) {
				errs = append(errs, ValidationError{
					Field: name,
					Err:   fmt.Errorf("%w: %q does not match %s", ErrRegexp, value, r.arg),
				})
			}
		case "in":
			if !slices.Contains(strings.Split(r.arg, ","), value) {
				errs = append(errs, ValidationError{
					Field: name,
					Err:   fmt.Errorf("%w: %q is not in [%s]", ErrNotInSet, value, r.arg),
				})
			}
		default:
			return nil, fmt.Errorf("field %s: %w: unknown rule %q", name, ErrInvalidTag, r.name)
		}
	}
	return errs, nil
}

func validateInt(fv reflect.Value, name, tag string) (ValidationErrors, error) {
	rules, err := parseRules(tag)
	if err != nil {
		return nil, fmt.Errorf("field %s: %w", name, err)
	}

	value := fv.Int()

	var errs ValidationErrors
	for _, r := range rules {
		switch r.name {
		case "min":
			limit, err := strconv.ParseInt(r.arg, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w: %q is not a number", name, ErrInvalidTag, r.arg)
			}
			if value < limit {
				errs = append(errs, ValidationError{
					Field: name,
					Err:   fmt.Errorf("%w: %d is less than %d", ErrTooSmall, value, limit),
				})
			}
		case "max":
			limit, err := strconv.ParseInt(r.arg, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w: %q is not a number", name, ErrInvalidTag, r.arg)
			}
			if value > limit {
				errs = append(errs, ValidationError{
					Field: name,
					Err:   fmt.Errorf("%w: %d is greater than %d", ErrTooBig, value, limit),
				})
			}
		case "in":
			options := strings.Split(r.arg, ",")
			nums := make([]int64, 0, len(options))
			for _, opt := range options {
				n, err := strconv.ParseInt(opt, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("field %s: %w: %q is not a number", name, ErrInvalidTag, opt)
				}
				nums = append(nums, n)
			}
			if !slices.Contains(nums, value) {
				errs = append(errs, ValidationError{
					Field: name,
					Err:   fmt.Errorf("%w: %d is not in [%s]", ErrNotInSet, value, r.arg),
				})
			}
		default:
			return nil, fmt.Errorf("field %s: %w: unknown rule %q", name, ErrInvalidTag, r.name)
		}
	}
	return errs, nil
}
