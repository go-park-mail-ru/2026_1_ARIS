package usecase

import "testing"

func TestIsMessageMediaSupportsLegacyAndFullMimeTypes(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		want     bool
	}{
		{name: "legacy image", mimeType: "image", want: true},
		{name: "full image", mimeType: "image/png", want: true},
		{name: "legacy video", mimeType: "video", want: true},
		{name: "full video", mimeType: "video/mp4", want: true},
		{name: "file", mimeType: "application/pdf", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMessageMedia(tt.mimeType); got != tt.want {
				t.Fatalf("isMessageMedia(%q) = %v, want %v", tt.mimeType, got, tt.want)
			}
		})
	}
}

func TestIsImageMediaSupportsLegacyAndFullMimeTypes(t *testing.T) {
	if !isImageMedia("image") {
		t.Fatal("legacy image mime type must be accepted")
	}
	if !isImageMedia("image/webp") {
		t.Fatal("full image mime type must be accepted")
	}
	if isImageMedia("video/mp4") {
		t.Fatal("video mime type must not be accepted as image")
	}
}
