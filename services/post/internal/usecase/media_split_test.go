package usecase

import "testing"

func TestIsTimelineMediaSupportsLegacyAndFullMimeTypes(t *testing.T) {
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
			if got := isTimelineMedia(tt.mimeType); got != tt.want {
				t.Fatalf("isTimelineMedia(%q) = %v, want %v", tt.mimeType, got, tt.want)
			}
		})
	}
}
