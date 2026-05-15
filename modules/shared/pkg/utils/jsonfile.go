// Package utils provides common utility functions.
package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

var ErrFileNotFound = errors.New("file not found")

// ReadJSONFile reads and unmarshals a JSON file into the target.
// Returns ErrFileNotFound if the file does not exist.
func ReadJSONFile(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		return fmt.Errorf("failed to read file: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	return nil
}

// ReadJSONFileOrDefault reads and unmarshals a JSON file.
// If the file doesn't exist or parsing fails, returns the default value.
// The defaultValue is deep-copied to target using JSON marshaling.
func ReadJSONFileOrDefault(path string, target interface{}, defaultValue interface{}) error {
	err := ReadJSONFile(path, target)
	if err != nil {
		// Copy default value to target using JSON marshal/unmarshal
		data, marshalErr := json.Marshal(defaultValue)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal default value: %w", marshalErr)
		}
		if unmarshalErr := json.Unmarshal(data, target); unmarshalErr != nil {
			return fmt.Errorf("failed to unmarshal default value: %w", unmarshalErr)
		}
	}
	return err
}

// WriteJSONFile atomically writes data to a JSON file.
// Uses a temporary file and rename for atomicity to prevent corruption.
func WriteJSONFile(path string, data interface{}) error {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Clean up temp file on rename failure
	if err := os.Rename(tmpFile, path); err != nil {
		os.Remove(tmpFile) // Ignore cleanup error
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// WriteJSONFileWithMode atomically writes data to a JSON file with custom permissions.
// Uses a temporary file and rename for atomicity to prevent corruption.
func WriteJSONFileWithMode(path string, data interface{}, perm os.FileMode) error {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, jsonBytes, perm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Clean up temp file on rename failure
	if err := os.Rename(tmpFile, path); err != nil {
		os.Remove(tmpFile) // Ignore cleanup error
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}
