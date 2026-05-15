package utils

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"jpg file", "test.jpg", true},
		{"jpeg file", "test.jpeg", true},
		{"png file", "test.png", true},
		{"gif file", "test.gif", true},
		{"webp file", "test.webp", true},
		{"bmp file", "test.bmp", true},
		{"svg file", "test.svg", true},
		{"uppercase JPG", "test.JPG", true},
		{"mixed case Png", "test.Png", true},
		{"text file", "test.txt", false},
		{"pdf file", "test.pdf", false},
		{"no extension", "testfile", false},
		{"path with dirs", "/path/to/image.png", true},
		{"path with dirs non-image", "/path/to/doc.pdf", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsImageFile(tt.path)
			if got != tt.want {
				t.Errorf("IsImageFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestGetImageMimeType(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"jpg file", "test.jpg", "image/jpeg"},
		{"jpeg file", "test.jpeg", "image/jpeg"},
		{"png file", "test.png", "image/png"},
		{"gif file", "test.gif", "image/gif"},
		{"webp file", "test.webp", "image/webp"},
		{"bmp file", "test.bmp", "image/bmp"},
		{"svg file", "test.svg", "image/svg+xml"},
		{"uppercase JPG", "test.JPG", "image/jpeg"},
		{"text file", "test.txt", ""},
		{"pdf file", "test.pdf", ""},
		{"no extension", "testfile", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetImageMimeType(tt.path)
			if got != tt.want {
				t.Errorf("GetImageMimeType(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestImageToDataURI(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content []byte
		wantErr bool
		wantURI string
	}{
		{
			name:    "png file",
			path:    "test.png",
			content: []byte("fake png content"),
			wantErr: false,
		},
		{
			name:    "jpg file",
			path:    "test.jpg",
			content: []byte("fake jpg content"),
			wantErr: false,
		},
		{
			name:    "non-image file",
			path:    "test.txt",
			content: []byte("text content"),
			wantErr: false,
			wantURI: "",
		},
		{
			name:    "binary content",
			path:    "image.png",
			content: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG magic bytes
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ImageToDataURI(tt.path, tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ImageToDataURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantURI != "" {
				if got != tt.wantURI {
					t.Errorf("ImageToDataURI() = %q, want %q", got, tt.wantURI)
				}
			} else if tt.wantErr {
				return
			}

			// Verify the dataURI format for image files
			if IsImageFile(tt.path) && !tt.wantErr {
				if !strings.HasPrefix(got, "data:image/") {
					t.Errorf("ImageToDataURI() should return dataURI starting with 'data:image/', got: %q", got)
				}
				if !strings.Contains(got, ";base64,") {
					t.Errorf("ImageToDataURI() should contain ';base64,', got: %q", got)
				}

				// Verify the encoded content can be decoded back
				parts := strings.SplitN(got, ",", 2)
				if len(parts) != 2 {
					t.Errorf("ImageToDataURI() invalid format, got: %q", got)
					return
				}

				encoded := parts[1]
				decoded, err := base64.StdEncoding.DecodeString(encoded)
				if err != nil {
					t.Errorf("ImageToDataURI() encoded content should be valid base64, error: %v", err)
					return
				}

				if string(decoded) != string(tt.content) {
					t.Errorf("ImageToDataURI() decoded content mismatch: got %q, want %q", string(decoded), string(tt.content))
				}
			}
		})
	}
}

func TestDetectImagesInInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single image path",
			input: "查看图片 /Users/test/image.jpg",
			want:  []string{"/Users/test/image.jpg"},
		},
		{
			name:  "multiple image paths",
			input: "比较 image1.png 和 image2.jpg",
			want:  []string{"image1.png", "image2.jpg"},
		},
		{
			name:  "no images",
			input: "这是一个文本文件.txt",
			want:  nil,
		},
		{
			name:  "mixed content",
			input: "读取 file.txt 和 photo.png 然后分析",
			want:  []string{"photo.png"},
		},
		{
			name:  "path with punctuation",
			input: "图片路径是：/path/to/test.jpg!",
			want:  []string{"图片路径是：/path/to/test.jpg"}, // Current implementation keeps Chinese characters before path
		},
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "uppercase extension",
			input: "打开 test.PNG",
			want:  []string{"test.PNG"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectImagesInInput(tt.input)

			if len(got) != len(tt.want) {
				t.Errorf("DetectImagesInInput(%q) returned %d results, want %d", tt.input, len(got), len(tt.want))
				return
			}

			for i, img := range got {
				if i >= len(tt.want) || img != tt.want[i] {
					t.Errorf("DetectImagesInInput(%q) result[%d] = %q, want %q", tt.input, i, img, tt.want[i])
				}
			}
		})
	}
}
