// Package utils provides utility functions for image processing.
package utils

import (
	"encoding/base64"
	"path/filepath"
	"strings"
)

// Supported image extensions
var imageExtensions = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".svg":  "image/svg+xml",
}

// IsImageFile checks if a file is an image based on its extension.
func IsImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := imageExtensions[ext]
	return ok
}

// GetImageMimeType returns the MIME type for an image file.
// Returns empty string if not an image.
func GetImageMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	return imageExtensions[ext]
}

// ImageToDataURI converts an image file to a dataURI string.
func ImageToDataURI(path string, content []byte) (string, error) {
	mimeType := GetImageMimeType(path)
	if mimeType == "" {
		return "", nil
	}

	encoded := base64.StdEncoding.EncodeToString(content)
	return "data:" + mimeType + ";base64," + encoded, nil
}

// DetectImagesInInput detects image file paths in a text input.
// Returns a list of detected image paths.
func DetectImagesInInput(input string) []string {
	// Simple detection: look for common image extensions in the input
	// This is a basic implementation - can be enhanced with regex later
	var images []string
	words := strings.Fields(input)
	for _, word := range words {
		// Clean common punctuation
		word = strings.Trim(word, ".,;:!?\"'()[]{}")
		if IsImageFile(word) {
			images = append(images, word)
		}
	}
	return images
}
