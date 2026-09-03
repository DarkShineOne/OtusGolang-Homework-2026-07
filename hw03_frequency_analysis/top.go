package hw03frequencyanalysis

import (
	"sort"
	"strings"
	"unicode"
)

type wordCount struct {
	word  string
	count int
}

const sliceLen = 10

func Top10(text string) []string {
	return getTopWords(text, false)
}

func Top10Normalized(text string) []string {
	return getTopWords(text, true)
}

func getTopWords(text string, normalize bool) []string {
	words := strings.Fields(text)
	freq := make(map[string]int)

	for _, word := range words {
		if normalize {
			word = normalizeWord(word)
			if word == "-" {
				continue
			}
		}

		freq[word]++
	}

	pairs := make([]wordCount, 0, len(freq))
	for word, count := range freq {
		pairs = append(pairs, wordCount{word, count})
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			return pairs[i].word < pairs[j].word
		}

		return pairs[i].count > pairs[j].count
	})

	maxLen := min(sliceLen, len(pairs))

	result := make([]string, 0, maxLen)

	for i := 0; i < maxLen; i++ {
		result = append(result, pairs[i].word)
	}

	return result
}

func normalizeWord(word string) string {
	word = strings.ToLower(word)
	word = strings.TrimFunc(word, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '-'
	})

	return word
}
