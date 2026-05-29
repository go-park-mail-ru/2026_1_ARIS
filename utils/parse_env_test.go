package utils

import (
	"testing"
	"time"
)

func TestEnvHelpers(t *testing.T) {
	t.Setenv("ARIS_TEST_INT", "42")
	t.Setenv("ARIS_TEST_BAD_INT", "bad")
	t.Setenv("ARIS_TEST_STRING", "value")
	t.Setenv("ARIS_TEST_BOOL", "true")
	t.Setenv("ARIS_TEST_DURATION", "2m")
	t.Setenv("ARIS_TEST_BAD_DURATION", "bad")
	t.Setenv("ARIS_TEST_BAD_BOOL", "bad")

	if EnvInt("ARIS_TEST_INT", 1) != 42 {
		t.Fatal("expected parsed int")
	}
	if EnvInt("ARIS_TEST_BAD_INT", 1) != 1 || EnvInt("ARIS_TEST_MISSING_INT", 2) != 2 {
		t.Fatal("expected int fallback")
	}
	if EnvString("ARIS_TEST_STRING", "fallback") != "value" || EnvString("ARIS_TEST_MISSING_STRING", "fallback") != "fallback" {
		t.Fatal("unexpected string env result")
	}
	if EnvDuration("ARIS_TEST_DURATION", time.Second) != 2*time.Minute {
		t.Fatal("expected parsed duration")
	}
	if EnvDuration("ARIS_TEST_BAD_DURATION", time.Second) != time.Second || EnvDuration("ARIS_TEST_MISSING_DURATION", 3*time.Second) != 3*time.Second {
		t.Fatal("expected duration fallback")
	}
	if !EnvBool("ARIS_TEST_BOOL", false) {
		t.Fatal("expected parsed bool")
	}
	if !EnvBool("ARIS_TEST_BAD_BOOL", true) || !EnvBool("ARIS_TEST_MISSING_BOOL", true) {
		t.Fatal("expected bool fallback")
	}
}
