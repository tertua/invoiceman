package controllers

import (
	"strings"
	"testing"
)

func TestResolveLocale(t *testing.T) {
	cases := [][2]string{{"id", "id"}, {"ID", "id"}, {" id ", "id"}, {"en", "en"}, {"EN", "en"}, {"", "en"}, {"fr", "en"}}
	for _, tc := range cases {
		if got := resolveLocale(tc[0]); got != tc[1] {
			t.Errorf("resolveLocale(%q) = %q, want %q", tc[0], got, tc[1])
		}
	}
}

func TestLanguageDirective(t *testing.T) {
	if got := languageDirective("id"); !strings.Contains(got, "Bahasa Indonesia") {
		t.Errorf("id directive missing Bahasa Indonesia: %q", got)
	}
	if got := languageDirective("en"); !strings.Contains(got, "English") {
		t.Errorf("en directive missing English: %q", got)
	}
}

func TestWriteNoteInstruction(t *testing.T) {
	terms := writeNoteInstruction("terms")
	if !strings.Contains(terms, "numbered list") || !strings.Contains(terms, "1.") {
		t.Errorf("terms instruction must require a numbered list: %q", terms)
	}
	if strings.Contains(terms, "wise saying") {
		t.Errorf("terms instruction must not use the notes style: %q", terms)
	}
	banned := []string{"thank you", "terima kasih", "kerja sama", "kerjasama", "cooperation"}
	for _, b := range banned {
		if !strings.Contains(terms, b) {
			t.Errorf("terms must forbid %q", b)
		}
	}
	note := writeNoteInstruction("description")
	if !strings.Contains(note, "wise saying") || !strings.Contains(note, "Do not summarize") {
		t.Errorf("description must request wise saying: %q", note)
	}
	for _, b := range banned {
		if !strings.Contains(note, b) {
			t.Errorf("note must forbid %q", b)
		}
	}
}

func TestCurrencyDirective(t *testing.T) {
	idr := currencyDirective("IDR")
	if !strings.Contains(idr, "Rp160.000") {
		t.Errorf("IDR directive missing Indonesian example: %q", idr)
	}
	if currencyDirective("") != "" {
		t.Error("empty currency must produce no directive")
	}
	if got := currencyDirective("usd"); !strings.Contains(got, "USD") {
		t.Errorf("code must be uppercased: %q", got)
	}
}
