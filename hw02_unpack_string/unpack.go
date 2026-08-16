package hw02unpackstring

import (
	"errors"
	"strings"
)

var ErrInvalidString = errors.New("invalid string")

const backslash = rune(0x5C)

func isASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func Unpack(s string) (string, error) {
	runes := []rune(s)

	var sb strings.Builder

	for i := 0; i < len(runes); {
		var ch rune

		switch {
		case runes[i] == backslash:
			if i+1 >= len(runes) {
				return "", ErrInvalidString
			}

			next := runes[i+1]

			if !isASCIIDigit(next) && next != backslash {
				return "", ErrInvalidString
			}

			ch = next
			i += 2

		case isASCIIDigit(runes[i]):
			return "", ErrInvalidString

		default:
			ch = runes[i]
			i++
		}

		count := 1
		if i < len(runes) && isASCIIDigit(runes[i]) {
			count = int(runes[i] - '0')
			i++
		}

		sb.WriteString(strings.Repeat(string(ch), count))
	}

	return sb.String(), nil
}
