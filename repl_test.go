package main

import (
	"testing"
)

func TestCleanInput(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{input: "  hello  world  ",
			expected: []string{"hello", "world"}},
		{input: "single",
			expected: []string{"single"}},
		{input: "Capital LetterS",
			expected: []string{"capital", "letters"}},
	}
	for _, c := range cases {
		actual := cleanInput(c.input)
		for i := range actual {
			word := actual[i]
			expectedWord := c.expected[i]
			if len(word) != len(expectedWord) {
				t.Errorf("got %v, want %v", word, expectedWord)
				if word != expectedWord {
					t.Errorf("words do not match, got %v and %v", word, expectedWord)
				}
			}
		}
	}
}
