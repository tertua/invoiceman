package configs

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// envOr returns the trimmed env value, or fallback when unset/blank.
func envOr(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}

// envInt returns the parsed env value or fallback when unset. A value that is
// set but not a positive integer is an error (D2): it is never silently
// swallowed, so a typo like RATE_LIMIT_GENERAL=0 fails startup loudly instead
// of falling back to a default. Callers that must not abort (Get) keep working
// because they ignore the error and get fallback.
func envInt(name string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback, fmt.Errorf("invalid %s %q: must be a positive integer", name, raw)
	}
	return v, nil
}

// envIntAllowZero parses an env value that may legitimately be zero (e.g.
// REDIS_DB_NUMBER). Set-but-unparsable values are collected for fail-fast
// while 0 stays valid.
func envIntAllowZero(name string, fallback int, issues *[]string) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		*issues = append(*issues, fmt.Sprintf("invalid %s %q: must be an integer", name, raw))
		return fallback
	}
	return v
}

// envBool returns the boolean env value; anything but true/false spellings
// falls back. Blank means unset.
func envBool(name string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	return strings.EqualFold(raw, "true")
}

// envList splits a comma-separated env value, dropping blanks.
func envList(name string) []string {
	var out []string
	for _, o := range strings.Split(strings.TrimSpace(os.Getenv(name)), ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}
