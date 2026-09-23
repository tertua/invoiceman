package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tertua/invoiceman/pkg/configs"
)

var ErrNotConfigured = errors.New("AI provider is not configured")

type GeminiClient struct {
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inline_data,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type geminiGenerationConfig struct {
	ResponseMimeType string  `json:"responseMimeType,omitempty"`
	Temperature      float64 `json:"temperature,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewGeminiClient() *GeminiClient {
	cfg := configs.Get().AI
	return &GeminiClient{
		APIKey:     cfg.GeminiKey,
		Model:      cfg.GeminiModel,
		HTTPClient: &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *GeminiClient) Generate(ctx context.Context, prompt string) (string, error) {
	return c.generate(ctx, []geminiPart{{Text: prompt}}, false)
}

func (c *GeminiClient) GenerateJSON(ctx context.Context, prompt string) (string, error) {
	return c.generate(ctx, []geminiPart{{Text: prompt}}, true)
}

func (c *GeminiClient) GenerateWithFile(ctx context.Context, prompt, mimeType string, data []byte) (string, error) {
	return c.generate(ctx, []geminiPart{
		{Text: prompt},
		{InlineData: &geminiInlineData{MimeType: mimeType, Data: base64.StdEncoding.EncodeToString(data)}},
	}, true)
}

func (c *GeminiClient) generate(ctx context.Context, parts []geminiPart, jsonResponse bool) (string, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return "", ErrNotConfigured
	}
	request := geminiRequest{Contents: []geminiContent{{Role: "user", Parts: parts}}}
	if jsonResponse {
		request.GenerationConfig = geminiGenerationConfig{ResponseMimeType: "application/json", Temperature: 0.2}
	}
	body, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.Model, c.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	var result geminiResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", fmt.Errorf("invalid Gemini response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if result.Error != nil && result.Error.Message != "" {
			return "", fmt.Errorf("gemini request failed: %s", result.Error.Message)
		}
		return "", fmt.Errorf("gemini request failed with status %d", resp.StatusCode)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("gemini returned no content")
	}
	return strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text), nil
}
