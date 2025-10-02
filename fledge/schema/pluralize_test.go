package schema

import "testing"

func TestPluralize(t *testing.T) {
	tests := []struct {
		singular string
		plural   string
	}{
		// Regular plurals (add s)
		{"cat", "cats"},
		{"dog", "dogs"},
		{"book", "books"},
		{"user", "users"},
		{"table", "tables"},

		// Words ending in s, x, z, ch, sh (add es)
		{"class", "classes"},
		{"box", "boxes"},
		{"quiz", "quizes"},
		{"church", "churches"},
		{"dish", "dishes"},
		{"bus", "buses"},
		{"fox", "foxes"},

		// Words ending in consonant + y (change to ies)
		{"city", "cities"},
		{"baby", "babies"},
		{"party", "parties"},
		{"story", "stories"},
		{"lady", "ladies"},

		// Words ending in vowel + y (just add s)
		{"boy", "boys"},
		{"key", "keys"},
		{"day", "days"},

		// Words ending in consonant + o (add es)
		{"hero", "heroes"},
		{"potato", "potatoes"},
		{"tomato", "tomatoes"},

		// Words ending in consonant + o exceptions (add s)
		{"photo", "photos"},
		{"piano", "pianos"},

		// Words ending in f or fe (change to ves)
		{"leaf", "leaves"},
		{"knife", "knives"},
		{"life", "lives"},
		{"wolf", "wolves"},
		{"wife", "wives"},

		// Irregular plurals
		{"person", "people"},
		{"child", "children"},
		{"man", "men"},
		{"woman", "women"},
		{"tooth", "teeth"},
		{"foot", "feet"},
		{"mouse", "mice"},
		{"goose", "geese"},

		// Case preservation
		{"User", "Users"},
		{"Person", "People"},
		{"CHILD", "CHILDREN"},

		// Edge cases
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.singular, func(t *testing.T) {
			result := Pluralize(tt.singular)
			if result != tt.plural {
				t.Errorf("Pluralize(%q) = %q; want %q", tt.singular, result, tt.plural)
			}
		})
	}
}

func TestIsVowel(t *testing.T) {
	tests := []struct {
		char     byte
		expected bool
	}{
		{'a', true},
		{'e', true},
		{'i', true},
		{'o', true},
		{'u', true},
		{'A', true},
		{'E', true},
		{'b', false},
		{'c', false},
		{'z', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			result := isVowel(tt.char)
			if result != tt.expected {
				t.Errorf("isVowel(%q) = %v; want %v", tt.char, result, tt.expected)
			}
		})
	}
}
