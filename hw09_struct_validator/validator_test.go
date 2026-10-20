package hw09structvalidator

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type UserRole string

// Test the function on different structures and other types.
type (
	User struct {
		ID     string `json:"id" validate:"len:36"`
		Name   string
		Age    int             `validate:"min:18|max:50"`
		Email  string          `validate:"regexp:^\\w+@\\w+\\.\\w+$"`
		Role   UserRole        `validate:"in:admin,stuff"`
		Phones []string        `validate:"len:11"`
		meta   json.RawMessage //nolint:unused
	}

	App struct {
		Version string `validate:"len:5"`
	}

	Token struct {
		Header    []byte
		Payload   []byte
		Signature []byte
	}

	Response struct {
		Code int    `validate:"in:200,404,500"`
		Body string `json:"omitempty"`
	}

	StringLen struct {
		S string `validate:"len:32"`
	}

	IntRange struct {
		N int `validate:"min:10|max:20"`
	}

	StringIn struct {
		S string `validate:"in:foo,bar"`
	}

	IntIn struct {
		N int `validate:"in:256,1024"`
	}

	StringRegexpLen struct {
		S string `validate:"regexp:\\d+|len:20"`
	}

	Order struct {
		Counts []int    `validate:"min:1|max:10"`
		Codes  []string `validate:"in:aa,bb"`
	}

	Meta struct {
		Name string `validate:"len:5"`
		Age  int    `validate:"min:10|max:20"`
	}

	Item struct {
		Meta Meta `validate:"nested"`
	}

	IgnoredNested struct {
		M Meta
	}

	Plain struct {
		A string
		b int //nolint:unused
	}

	BadTagMin struct {
		N int `validate:"min:abc"`
	}

	BadTagUnknown struct {
		N int `validate:"unknown:5"`
	}

	BadTagNoArg struct {
		S string `validate:"len"`
	}

	BadTagRegexp struct {
		S string `validate:"regexp:(("`
	}

	BoolTag struct {
		B bool `validate:"min:1"`
	}

	BytesTag struct {
		B []byte `validate:"len:5"`
	}
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		// wantErrs - ожидаемые ошибки валидации (ValidationErrors),
		// в порядке объявления полей структуры.
		wantErrs []ValidationError
		// wantProgErr - ожидаемая программная ошибка (не ValidationErrors).
		wantProgErr error
	}{
		{
			name:        "nil is not a struct",
			in:          nil,
			wantProgErr: ErrNotStruct,
		},
		{
			name:        "int is not a struct",
			in:          42,
			wantProgErr: ErrNotStruct,
		},
		{
			name:        "string is not a struct",
			in:          "hello",
			wantProgErr: ErrNotStruct,
		},
		{
			name:        "map is not a struct",
			in:          map[string]int{"a": 1},
			wantProgErr: ErrNotStruct,
		},
		{
			name:        "slice is not a struct",
			in:          []int{1, 2, 3},
			wantProgErr: ErrNotStruct,
		},
		{
			name:        "pointer to non-struct is not a struct",
			in:          new(int),
			wantProgErr: ErrNotStruct,
		},
		{
			name: "fields without validate tag are ignored",
			in:   Token{Header: []byte("h"), Payload: nil, Signature: nil},
		},
		{
			name: "unexported and untagged fields are ignored",
			in:   Plain{A: "whatever"},
		},
		{
			name: "valid App",
			in:   App{Version: "1.0.0"},
		},
		{
			name:     "App version wrong length",
			in:       App{Version: "1.0"},
			wantErrs: []ValidationError{{Field: "Version", Err: ErrLen}},
		},
		{
			name: "string len exact match",
			in:   StringLen{S: strings.Repeat("a", 32)},
		},
		{
			name:     "string len too short",
			in:       StringLen{S: strings.Repeat("a", 31)},
			wantErrs: []ValidationError{{Field: "S", Err: ErrLen}},
		},
		{
			name:     "string len too long",
			in:       StringLen{S: strings.Repeat("a", 33)},
			wantErrs: []ValidationError{{Field: "S", Err: ErrLen}},
		},
		{
			name: "int min boundary",
			in:   IntRange{N: 10},
		},
		{
			name: "int max boundary",
			in:   IntRange{N: 20},
		},
		{
			name:     "int below min",
			in:       IntRange{N: 9},
			wantErrs: []ValidationError{{Field: "N", Err: ErrTooSmall}},
		},
		{
			name:     "int above max",
			in:       IntRange{N: 21},
			wantErrs: []ValidationError{{Field: "N", Err: ErrTooBig}},
		},
		{
			name: "string in set",
			in:   StringIn{S: "foo"},
		},
		{
			name:     "string not in set",
			in:       StringIn{S: "baz"},
			wantErrs: []ValidationError{{Field: "S", Err: ErrNotInSet}},
		},
		{
			name: "int in set",
			in:   IntIn{N: 1024},
		},
		{
			name:     "int not in set",
			in:       IntIn{N: 512},
			wantErrs: []ValidationError{{Field: "N", Err: ErrNotInSet}},
		},
		{
			name: "combined regexp and len - valid",
			in:   StringRegexpLen{S: strings.Repeat("7", 20)},
		},
		{
			name:     "combined regexp and len - len fails only",
			in:       StringRegexpLen{S: strings.Repeat("7", 19)},
			wantErrs: []ValidationError{{Field: "S", Err: ErrLen}},
		},
		{
			name:     "combined regexp and len - both fail, errors accumulated",
			in:       StringRegexpLen{S: strings.Repeat("a", 19)},
			wantErrs: []ValidationError{{Field: "S", Err: ErrRegexp}, {Field: "S", Err: ErrLen}},
		},
		{
			name:  "valid Order",
			in:    Order{Counts: []int{1, 5, 10}, Codes: []string{"aa", "bb"}},
		},
		{
			name: "empty slices are valid",
			in:   Order{Counts: []int{}, Codes: []string{}},
		},
		{
			name: "nil slices are valid",
			in:   Order{},
		},
		{
			name: "slice elements validated one by one",
			in:   Order{Counts: []int{5, 0, 11}},
			wantErrs: []ValidationError{
				{Field: "Counts", Err: ErrTooSmall},
				{Field: "Counts", Err: ErrTooBig},
			},
		},
		{
			name: "errors from several fields are accumulated",
			in:   Order{Counts: []int{0}, Codes: []string{"zz"}},
			wantErrs: []ValidationError{
				{Field: "Counts", Err: ErrTooSmall},
				{Field: "Codes", Err: ErrNotInSet},
			},
		},
		{
			name: "valid User",
			in: User{
				ID:     strings.Repeat("a", 36),
				Name:   "any name",
				Age:    30,
				Email:  "user@mail.com",
				Role:   "stuff",
				Phones: []string{"79998887766"},
			},
		},
		{
			name: "User with several invalid fields",
			in: User{
				ID:     "short",
				Name:   "any name",
				Age:    17,
				Email:  "bad",
				Role:   "admin",
				Phones: []string{"79998887766"},
			},
			wantErrs: []ValidationError{
				{Field: "ID", Err: ErrLen},
				{Field: "Age", Err: ErrTooSmall},
				{Field: "Email", Err: ErrRegexp},
			},
		},
		{
			name: "valid Response",
			in:   Response{Code: 404, Body: "not found"},
		},
		{
			name:     "Response code not in set",
			in:       Response{Code: 301},
			wantErrs: []ValidationError{{Field: "Code", Err: ErrNotInSet}},
		},
		{
			name:        "invalid tag value",
			in:          BadTagMin{N: 1},
			wantProgErr: ErrInvalidTag,
		},
		{
			name:        "unknown validator in tag",
			in:          BadTagUnknown{N: 1},
			wantProgErr: ErrInvalidTag,
		},
		{
			name:        "tag without argument",
			in:          BadTagNoArg{S: "x"},
			wantProgErr: ErrInvalidTag,
		},
		{
			name:        "invalid regexp in tag",
			in:          BadTagRegexp{S: "x"},
			wantProgErr: ErrInvalidTag,
		},
		{
			name:        "unsupported type bool",
			in:          BoolTag{B: true},
			wantProgErr: ErrUnsupportedType,
		},
		{
			name:        "unsupported type slice of bytes",
			in:          BytesTag{B: []byte("12345")},
			wantProgErr: ErrUnsupportedType,
		},
		{
			name: "valid nested struct",
			in:   Item{Meta: Meta{Name: "abcde", Age: 15}},
		},
		{
			name: "invalid nested struct",
			in:   Item{Meta: Meta{Name: "x", Age: 9}},
			wantErrs: []ValidationError{
				{Field: "Meta.Name", Err: ErrLen},
				{Field: "Meta.Age", Err: ErrTooSmall},
			},
		},
		{
			name: "nested struct without nested tag is ignored",
			in:   IgnoredNested{M: Meta{Name: "x", Age: 999}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Validate(tt.in)

			switch {
			case tt.wantProgErr != nil:
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantProgErr)
				var verrs ValidationErrors
				assert.False(t, errors.As(err, &verrs), "programmatic error must not be ValidationErrors")
			case tt.wantErrs != nil:
				require.Error(t, err)
				var verrs ValidationErrors
				require.ErrorAs(t, err, &verrs)
				require.Len(t, verrs, len(tt.wantErrs))
				for i := range tt.wantErrs {
					assert.Equal(t, tt.wantErrs[i].Field, verrs[i].Field, "unexpected field at index %d", i)
					assert.ErrorIs(t, verrs[i].Err, tt.wantErrs[i].Err, "unexpected error for field %s", tt.wantErrs[i].Field)
				}
			default:
				require.NoError(t, err)
			}
		})
	}
}

func TestValidationErrorsError(t *testing.T) {
	errs := ValidationErrors{
		{Field: "Name", Err: ErrLen},
		{Field: "Age", Err: ErrTooSmall},
	}

	var err error = errs
	require.NotPanics(t, func() { _ = err.Error() })

	msg := err.Error()
	assert.Contains(t, msg, "Name")
	assert.Contains(t, msg, "Age")
	assert.Contains(t, msg, ErrLen.Error())
	assert.Contains(t, msg, ErrTooSmall.Error())
}
