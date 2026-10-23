package main

import "testing"

func TestSponsorLanguages(t *testing.T) {
	for code := range languages {
		row, ok := sponsorTranslations[code]
		if !ok {
			t.Errorf("missing sponsor translation: %s", code)
			continue
		}
		for i, text := range row {
			if text == "" {
				t.Errorf("empty %s field %d", code, i)
			}
			if sponsorLocalized(code, i) != text {
				t.Errorf("incorrect %s field %d", code, i)
			}
			if code != "en" && text == sponsorTranslations["en"][i] {
				t.Errorf("English fallback for %s field %d", code, i)
			}
		}
	}
	if sponsorLocalized("unknown", 0) != sponsorTranslations["en"][0] {
		t.Fatal("missing fallback")
	}
}
