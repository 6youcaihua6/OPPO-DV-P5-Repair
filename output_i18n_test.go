package main

import (
	"strings"
	"testing"
)

func TestOutputNoteEffectiveTranslations(t *testing.T) {
	seen := map[string]string{}
	for code := range languages {
		expected, ok := outputNotes[code]
		if !ok || expected == "" {
			t.Errorf("missing output note: %s", code)
			continue
		}
		actual := tr(code, "outputNote")
		if actual != expected {
			t.Errorf("%s output note overwritten: %q", code, actual)
		}
		if other, duplicate := seen[actual]; duplicate {
			t.Errorf("%s and %s share output note", code, other)
		}
		seen[actual] = code
		for _, folder := range []string{"Videos", "Reports"} {
			if !strings.Contains(actual, folder) {
				t.Errorf("%s missing literal folder %s", code, folder)
			}
		}
	}
}
