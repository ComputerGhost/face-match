package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"
)

type errorResponse struct {
	Detail any `json:"detail"`
}

type EmbeddingOkResponse struct {
	Embedding []float64 `json:"embedding"`
	Dim       int       `json:"dim"`
	BBox      []float64 `json:"bbox"`
	DetScore  *float64  `json:"det_score"`
}

func FetchEmbedding(ctx context.Context, endpoint string, imageBytes []byte) ([]float32, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("FetchEmbedding: AIEndpoint is required")
	}
	if len(imageBytes) == 0 {
		return nil, fmt.Errorf("FetchEmbedding: empty image")
	}

	url := endpoint
	// tolerate trailing slash
	if url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	url += "/embed-largest-face"

	request, err := buildRequest(ctx, url, imageBytes)
	if err != nil {
		return nil, fmt.Errorf("FetchEmbedding: error building request: %v", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("FetchEmbedding: request failed: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	result, err := processResponse(response)
	if err != nil {
		return nil, fmt.Errorf("FetchEmbedding: error processing response: %v", err)
	}

	embedding := make([]float32, len(result.Embedding))
	for i, v := range result.Embedding {
		embedding[i] = float32(v)
	}
	return embedding, nil
}

func buildRequest(ctx context.Context, url string, imageBytes []byte) (*http.Request, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "upload.jpg")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(imageBytes); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return nil, fmt.Errorf("FetchEmbedding: new request: %w", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request, nil
}

func processResponse(response *http.Response) (*EmbeddingOkResponse, error) {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var er errorResponse
		_ = json.NewDecoder(response.Body).Decode(&er)
		if er.Detail != nil {
			return nil, fmt.Errorf("processResponse: sidecar error (%d): %v", response.StatusCode, er.Detail)
		}
		return nil, fmt.Errorf("processResponse: sidecar error (%d): %s", response.StatusCode, response.Status)
	}

	var result EmbeddingOkResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("processResponse: decode response: %w", err)
	}
	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("processResponse: empty embedding returned")
	}
	if result.Dim != 0 && result.Dim != len(result.Embedding) {
		return nil, fmt.Errorf("processResponse: dim mismatch: dim=%d len=%d", result.Dim, len(result.Embedding))
	}
	return &result, nil
}
