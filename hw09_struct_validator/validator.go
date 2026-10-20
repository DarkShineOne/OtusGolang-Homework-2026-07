package hw09structvalidator

import (
	"errors"
	"reflect"
)

type ValidationError struct {
	Field string
	Err   error
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	panic("implement me")
}

func Validate(v interface{}) error {
	reflectVal := reflect.ValueOf(v)
	reflectType := reflectVal.Type()

	if reflectVal.Kind() != reflect.Struct {
		return errors.New("v must be a struct")
	}

	for i := 0; i < reflectVal.NumField(); i++ {
		needValidate := reflectType.Field(i).Tag.Get("validate")
		if needValidate == "" {
			continue
		}

		switch reflectVal.Field(i).Kind() {
		case reflect.Struct:
			break
		case reflect.Int:
			err := intValidator(i)
			if err != nil {
				return err
			}
		case reflect.String:
			err := stringValidator(i)
			if err != nil {
				return err
			}
		default:
		}
	}

	return nil
}

func stringValidator(field string) error {

}

func intValidator(field int) error {

}
