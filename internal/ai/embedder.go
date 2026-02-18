package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
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
	DetScore  float64   `json:"det_score"`
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
	if _, err := isFoundFaceGood(result, imageBytes); err != nil {
		return nil, fmt.Errorf("FetchEmbedding: error checking face: %v", err)
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

func isFoundFaceGood(result *EmbeddingOkResponse, imageBytes []byte) (bool, error) {
	if result.DetScore < 0.5 {
		return false, fmt.Errorf("det score of %f is too low", result.DetScore)
	}

	faceHeight := result.BBox[3] - result.BBox[1]
	if faceHeight < 92 {
		return false, fmt.Errorf("face height of %f is too small", faceHeight)
	}

	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return false, fmt.Errorf("decode: %v", err)
	}
	b := img.Bounds()
	if b.Empty() {
		return false, fmt.Errorf("empty image")
	}

	// Clamp bbox to image bounds
	x1 := int(math.Floor(result.BBox[0]))
	y1 := int(math.Floor(result.BBox[1]))
	x2 := int(math.Ceil(result.BBox[2]))
	y2 := int(math.Ceil(result.BBox[3]))
	if x1 < b.Min.X {
		x1 = b.Min.X
	}
	if y1 < b.Min.Y {
		y1 = b.Min.Y
	}
	if x2 > b.Max.X {
		x2 = b.Max.X
	}
	if y2 > b.Max.Y {
		y2 = b.Max.Y
	}

	if x2-x1 < 8 || y2-y1 < 8 {
		// Too small to blur meaningfully
		return false, fmt.Errorf("face crop too small (%dx%d)", x2-x1, y2-y1)
	}

	crop := image.Rect(x1, y1, x2, y2)

	w := crop.Dx()
	h := crop.Dy()
	gray := make([]float64, w*h)

	i := 0
	for y := crop.Min.Y; y < crop.Max.Y; y++ {
		for x := crop.Min.X; x < crop.Max.X; x++ {
			gray[i] = float64(luma8(img.At(x, y)))
			i++
		}
	}

	var sum, sumSq float64
	for y := 1; y < h-1; y++ {
		row := y * w
		for x := 1; x < w-1; x++ {
			c := gray[row+x]
			l := gray[row+x-1]
			r := gray[row+x+1]
			u := gray[row-w+x]
			d := gray[row+w+x]
			lap := (-4.0 * c) + l + r + u + d
			sum += lap
			sumSq += lap * lap
		}
	}
	mean := sum / float64(h*w)
	variance := (sumSq / float64(h*w)) - (mean * mean)

	if variance < 20 {
		return false, fmt.Errorf("variance %f is too low", variance)
	}

	return true, nil
}

func luma8(c color.Color) uint8 {
	r, g, b, _ := c.RGBA()
	R := uint32(r >> 8)
	G := uint32(g >> 8)
	B := uint32(b >> 8)
	Y := (299*R + 587*G + 114*B + 500) / 1000
	return uint8(Y)
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
