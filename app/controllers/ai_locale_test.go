package controllers

import (
	"strings"
	"testing"
)

// resolveLocale allowlists the SPA language header.
func TestResolveLocale(t *testing.T) {
	cases := [][2]string{
		{"id", "id"},
		{"ID", "id"},
		{" id ", "id"},
		{"en", "en"},
		{"EN", "en"},
		{"", "en"},
		{"fr", "en"},
	}
	for _, tc := range cases {
		if got := resolveLocale(tc[0]); got != tc[1] {
			t.Errorf("resolveLocale(%q) = %q, want %q", tc[0], got, tc[1])
		}
	}
}

// languageDirective must pin a language explicitly: bare money data would
// otherwise pull the answer (and its currency formatting) toward English.
func TestLanguageDirective(t *testing.T) {
	if got := languageDirective("id"); !strings.Contains(got, "Bahasa Indonesia") {
		t.Errorf("id directive missing Bahasa Indonesia: %q", got)
	}
	if got := languageDirective("en"); !strings.Contains(got, "English") {
		t.Errorf("en directive missing English: %q", got)
	}
}

// writeNoteInstruction turns notes into a wise saying and terms into a numbered-only list.
func TestWriteNoteInstruction(t *testing.T) {
	terms := writeNoteInstruction("terms")
	if !strings.Contains(terms, "numbered list") || !strings.Contains(terms, "1.") {
		t.Errorf("terms instruction must require a numbered list: %q", terms)
	}
	if strings.Contains(terms, "wise saying") {
		t.Errorf("terms instruction must not use the notes style: %q", terms)
	}
	if got := writeNoteInstruction("description"); !strings.Contains(got, "wise saying") || !strings.Contains(got, "Do not summarize") {
		t.Errorf("description instruction must request a wise saying, not a summary: %q", got)
	}
}

// currencyDirective must bind formatting to the ISO code so the model never
// defaults bare numbers to US dollars.
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
