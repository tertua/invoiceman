# AI and Localization

## Rules
- AI locale from X-Locale header (en/id), default en when missing or unknown
- Language directive pins answer language explicitly ("Respond entirely in Bahasa Indonesia")
- Currency directive prevents $ defaults (currencyDirective binds to ISO code)
- AI errors logged server-side, generic message to client (no provider details leak)
- Gemini 2.0-flash default model, configurable via GEMINI_API_KEY env
- AI text cached in localStorage per user+invoice+tone+language
- Regenerate overwrites cache instead of re-calling API
- Rate limit 429 → HTTP 429 (ai.rateLimited)

## Examples

### Locale Resolution
```go
// app/controllers/ai_controller.go:13-27
func aiLocale(c fiber.Ctx) string {
	return resolveLocale(c.Get("X-Locale"))
}

func resolveLocale(header string) string {
	if strings.EqualFold(strings.TrimSpace(header), "id") {
		return "id"
	}
	return "en"
}

func languageDirective(lang string) string {
	if lang == "id" {
		return "Respond entirely in Bahasa Indonesia. "
	}
	return "Respond entirely in English. "
}
```

### Currency Directive (Prevents $ Default)
```go
// app/controllers/ai_controller.go:35-46
func currencyDirective(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return ""
	}
	if code == "IDR" {
		return "All money amounts are in Indonesian rupiah (IDR). Write them Indonesian style (for example Rp160.000 with dots as thousand separators) and never use $. "
	}
	return "All money amounts are in " + code + ". Write them in that currency's conventional format and never default to US dollars. "
}
```

### Frontend X-Locale Header Injection
```js
// webui/src/api/http.js:47-61
apiClient.interceptors.request.use((config) => {
	config.headers = config.headers ?? {};
	const token = csrfToken();
	if (token) config.headers["X-CSRF-Token"] = token;
	// UI language for endpoints that generate text (AI): same localStorage
	// key LangContext persists, allowlisted so a stale value can't leak through.
	config.headers["X-Locale"] = appLocale();
	return config;
});

function appLocale() {
	try {
		return localStorage.getItem("arr-lang") === "id" ? "id" : "en";
	} catch {
		return "en";
	}
}
```

## Anti-patterns

### ❌ Bare Money in AI Prompts
```go
// BAD: AI defaults to USD
prompt := "Calculate invoice total: " + amount

// GOOD: currency directive
prompt := currencyDirective("IDR") + "Calculate invoice total: " + amount
```

### ❌ Missing Language Directive
```go
// BAD: AI might respond in English for Indonesian user
prompt := "Write invoice terms"

// GOOD: explicit language
prompt := languageDirective(aiLocale(c)) + "Write invoice terms"
```
