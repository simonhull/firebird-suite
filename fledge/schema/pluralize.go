package schema

import (
	"strings"
	"unicode"
)

// Pluralize converts singular nouns to plural form using common English rules
func Pluralize(word string) string {
	if word == "" {
		return ""
	}

	lower := strings.ToLower(word)

	// Irregular plurals
	irregulars := map[string]string{
		"person": "people",
		"child":  "children",
		"man":    "men",
		"woman":  "women",
		"tooth":  "teeth",
		"foot":   "feet",
		"mouse":  "mice",
		"goose":  "geese",
	}
	if plural, ok := irregulars[lower]; ok {
		return preserveCase(word, plural)
	}

	// Words ending in s, x, z, ch, sh: add "es"
	if strings.HasSuffix(lower, "s") ||
		strings.HasSuffix(lower, "x") ||
		strings.HasSuffix(lower, "z") ||
		strings.HasSuffix(lower, "ch") ||
		strings.HasSuffix(lower, "sh") {
		return word + "es"
	}

	// Words ending in consonant + y: change y to ies
	if strings.HasSuffix(lower, "y") && len(word) > 1 {
		beforeY := lower[len(lower)-2]
		if !isVowel(beforeY) {
			return word[:len(word)-1] + "ies"
		}
	}

	// Words ending in consonant + o: add "es" (with exceptions)
	if strings.HasSuffix(lower, "o") && len(word) > 1 {
		beforeO := lower[len(lower)-2]
		if !isVowel(beforeO) {
			// Common exceptions that just add "s"
			exceptions := []string{"photo", "piano", "halo"}
			for _, exc := range exceptions {
				if strings.HasSuffix(lower, exc) {
					return word + "s"
				}
			}
			return word + "es"
		}
	}

	// Words ending in f or fe: change to ves
	if strings.HasSuffix(lower, "f") {
		return word[:len(word)-1] + "ves"
	}
	if strings.HasSuffix(lower, "fe") {
		return word[:len(word)-2] + "ves"
	}

	// Default: just add "s"
	return word + "s"
}

// preserveCase applies the case pattern from original to the plural form
func preserveCase(original, plural string) string {
	if len(original) == 0 {
		return plural
	}

	// All uppercase
	if strings.ToUpper(original) == original {
		return strings.ToUpper(plural)
	}

	// Title case (first letter uppercase)
	if unicode.IsUpper(rune(original[0])) {
		return strings.ToUpper(plural[:1]) + plural[1:]
	}

	return plural
}

// isVowel checks if a byte represents a vowel
func isVowel(c byte) bool {
	c = byte(strings.ToLower(string(c))[0])
	return c == 'a' || c == 'e' || c == 'i' || c == 'o' || c == 'u'
}
