package nirikshaai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GetPromptOptions configures a prompt fetch.
type GetPromptOptions struct {
	Version   *int
	Variables map[string]string
}

// GetPrompt fetches and renders a prompt template from the NirikshaAI prompt vault.
// The project is resolved from the API key — no org/project IDs needed.
func GetPrompt(ctx context.Context, name string, opts *GetPromptOptions) (string, error) {
	payload := map[string]any{"name": name, "variables": map[string]string{}}
	if opts != nil {
		if opts.Variables != nil {
			payload["variables"] = opts.Variables
		}
		if opts.Version != nil {
			payload["version"] = *opts.Version
		}
	}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		_state.baseURL+"/api/v1/sdk/prompts/render", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", _state.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("nirikshaai: get_prompt status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var out struct {
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	return out.Data.Content, nil
}

// PromptInfo is a brief summary of a stored prompt template.
type PromptInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// ListPrompts lists all prompt templates for the project.
func ListPrompts(ctx context.Context) ([]PromptInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		_state.baseURL+"/api/v1/sdk/prompts", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", _state.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out struct {
		Data struct {
			Prompts []PromptInfo `json:"prompts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out.Data.Prompts, nil
}
