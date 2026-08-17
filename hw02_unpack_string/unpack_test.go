package hw02unpackstring

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnpack(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "a4bc2d5e", expected: "aaaabccddddde"},
		{input: "abccd", expected: "abccd"},
		{input: "", expected: ""},
		{input: "aaa0b", expected: "aab"},
		{input: "🙃0", expected: ""},
		{input: "aaф0b", expected: "aab"},
		{input: "αβγ2", expected: "αβγγ"},
		{input: "🎉2🎊3", expected: "🎉🎉🎊🎊🎊"},
		{input: "a", expected: "a"},
		{input: "a9", expected: "aaaaaaaaa"},
		{input: `qwe\4\5`, expected: `qwe45`},
		{input: "的3", expected: "的的的"},
		{input: `qwe\45`, expected: `qwe44444`},
		{input: `qwe\\5`, expected: `qwe\\\\\`},
		{input: `qwe\\\3`, expected: `qwe\3`},
		{input: `的\3`, expected: `的3`},
		{input: `a\1`, expected: `a1`},
		{input: `\5a`, expected: `5a`},
		{input: `a\5`, expected: `a5`},
		{input: "a٥a", expected: "a٥a"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result, err := Unpack(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestUnpackInvalidString(t *testing.T) {
	invalidStrings := []string{
		"3abc",
		"45",
		"aaa10b",
		"qw\\ne",
		"a\\",
		"\\",
		"\\a",
		"a\\b",
	}
	for _, tc := range invalidStrings {
		t.Run(tc, func(t *testing.T) {
			_, err := Unpack(tc)
			require.Truef(t, errors.Is(err, ErrInvalidString), "actual error %q", err)
		})
	}
}
