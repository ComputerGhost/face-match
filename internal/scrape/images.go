package scrape

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gocolly/colly"
)

func IsImage(r *colly.Response) bool {
	return strings.Index(r.Headers.Get("Content-Type"), "image") > -1
}

func SaveImage(r *colly.Response, outDir string, filename string) error {
	extension := filepath.Ext(r.FileName())
	outPath := filepath.Join(outDir, sanitize(filename+extension))
	fmt.Println("Saving image", filename)
	return r.Save(outPath)
}

func sanitize(filename string) string {
	invalidChars := "/\\:*?<>|"
	sanitized := filename
	for _, char := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, string(char), "_")
	}
	return sanitized
}
