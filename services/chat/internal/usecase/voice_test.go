package usecase

import "testing"

func TestIsMessageMediaIncludesAudio(t *testing.T) {
	t.Parallel()

	if !isMessageMedia("audio/webm") {
		t.Fatal("audio/webm should be rendered as chat media")
	}
	if !isMessageMedia("audio/mp4") {
		t.Fatal("audio/mp4 should be rendered as chat media")
	}
}
