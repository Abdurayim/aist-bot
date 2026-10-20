package validator

import "testing"

func TestValidatePhone(t *testing.T) {
	valid := map[string]string{
		"+998901234567":     "+998901234567",
		"998901234567":      "+998901234567",
		"901234567":         "+998901234567",
		"+998 90 123 45 67": "+998901234567",
		"+998-90-123-45-67": "+998901234567",
	}

	for input, want := range valid {
		got, err := ValidatePhone(input)
		if err != nil {
			t.Errorf("ValidatePhone(%q) returned an error: %v", input, err)
			continue
		}

		if got != want {
			t.Errorf("ValidatePhone(%q) = %q, want %q", input, got, want)
		}
	}

	invalid := []string{"", "12345", "+7999123456", "abcdefghi", "+99890123456789"}

	for _, input := range invalid {
		if _, err := ValidatePhone(input); err == nil {
			t.Errorf("ValidatePhone(%q) should have failed", input)
		}
	}
}

func TestValidateText(t *testing.T) {
	if _, _, err := ValidateText("qisqa"); err == nil {
		t.Error("a too-short text should be rejected")
	}

	long := make([]rune, MaxTextLength+1)
	for i := range long {
		long[i] = 'a'
	}

	if _, _, err := ValidateText(string(long)); err == nil {
		t.Error("a too-long text should be rejected")
	}

	// Cyrillic is counted in runes, not bytes, so a message well under the
	// limit in characters is not rejected for being over it in bytes.
	cyrillic := make([]rune, MaxTextLength-1)
	for i := range cyrillic {
		cyrillic[i] = 'я'
	}

	if _, _, err := ValidateText(string(cyrillic)); err != nil {
		t.Errorf("a Cyrillic message within the limit was rejected: %v", err)
	}

	got, _, err := ValidateText("  Oshxonadagi ovqat sovuq.  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "Oshxonadagi ovqat sovuq." {
		t.Errorf("text was not trimmed: %q", got)
	}
}

func TestValidateChildName(t *testing.T) {
	good := map[string]string{
		"Aliyev Sardor":   "Aliyev Sardor",
		"  Алиев Сардор ": "Алиев Сардор",
		"Aliyev   Sardor": "Aliyev Sardor",
	}

	for input, want := range good {
		got, err := ValidateChildName(input)
		if err != nil {
			t.Errorf("ValidateChildName(%q) returned an error: %v", input, err)
			continue
		}

		if got != want {
			t.Errorf("ValidateChildName(%q) = %q, want %q", input, got, want)
		}
	}

	for _, input := range []string{"", ".", "ab", "12", "..."} {
		if _, err := ValidateChildName(input); err == nil {
			t.Errorf("ValidateChildName(%q) should have failed", input)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := map[string]string{
		"Aliyev Sardor":    "Aliyev_Sardor",
		"Алиев Сардор":     "murojaat",
		"Ali/../../ev":     "Aliev",
		"AIST-2026-000042": "AIST-2026-000042",
	}

	for input, want := range cases {
		if got := SanitizeFilename(input); got != want {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", input, got, want)
		}
	}
}
