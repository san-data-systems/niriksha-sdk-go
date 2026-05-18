package nirikshaai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// PromptResponse holds a fully rendered prompt and its metadata.
type PromptResponse struct {
	Content   string    // rendered prompt text
	Tags      []string  // categorisation tags
	CreatedAt time.Time // when this prompt was created
	UpdatedAt time.Time // when this version was last updated
}

type _cachedPrompt struct {
	resp      PromptResponse
	expiresAt time.Time
}

var _promptCache sync.Map

var _promptClient = &http.Client{Timeout: 10 * time.Second}

// GetPromptOptions configures a prompt fetch.
type GetPromptOptions struct {
	Version   *int
	Variables map[string]string
}

// GetPrompt fetches and renders a prompt template from the NirikshaAI prompt vault.
// The project is resolved from the API key — no org/project IDs needed.
// Results are cached in memory for 5 minutes.
func GetPrompt(ctx context.Context, name string, opts *GetPromptOptions) (string, error) {
	resp, err := GetPromptFull(ctx, name, opts)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// GetPromptFull fetches and renders a prompt template, returning the full
// PromptResponse including metadata. Results are cached in memory for 5 minutes.
func GetPromptFull(ctx context.Context, name string, opts *GetPromptOptions) (PromptResponse, error) {
	// Build cache key.
	version := 0
	if opts != nil && opts.Version != nil {
		version = *opts.Version
	}
	versionKey := "latest"
	if version != 0 {
		versionKey = strconv.Itoa(version)
	}
	cacheKey := name + ":" + versionKey

	// Check cache.
	if v, ok := _promptCache.Load(cacheKey); ok {
		entry := v.(_cachedPrompt)
		if time.Now().Before(entry.expiresAt) {
			return entry.resp, nil
		}
		_promptCache.Delete(cacheKey)
	}

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
		log.Printf("nirikshaai: get_prompt build request error: %v", err)
		return PromptResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", _state.apiKey)

	resp, err := _promptClient.Do(req)
	if err != nil {
		log.Printf("nirikshaai: get_prompt request error: %v", err)
		return PromptResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		err = fmt.Errorf("nirikshaai: get_prompt status %d", resp.StatusCode)
		log.Printf("nirikshaai: %v", err)
		return PromptResponse{}, err
	}

	body, _ := io.ReadAll(resp.Body)
	var out struct {
		Data struct {
			Content   string   `json:"content"`
			Tags      []string `json:"tags"`
			CreatedAt string   `json:"created_at"`
			UpdatedAt string   `json:"updated_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		log.Printf("nirikshaai: get_prompt parse error: %v", err)
		return PromptResponse{}, err
	}

	result := PromptResponse{
		Content: out.Data.Content,
		Tags:    out.Data.Tags,
	}
	if out.Data.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, out.Data.CreatedAt); err == nil {
			result.CreatedAt = t
		}
	}
	if out.Data.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, out.Data.UpdatedAt); err == nil {
			result.UpdatedAt = t
		}
	}

	_promptCache.Store(cacheKey, _cachedPrompt{
		resp:      result,
		expiresAt: time.Now().Add(5 * time.Minute),
	})

	return result, nil
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
	resp, err := _promptClient.Do(req)
	if err != nil {
		log.Printf("nirikshaai: list_prompts request error: %v", err)
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
