package utils

import "testing"

func TestEnvHelpers(t *testing.T) {
	t.Setenv("ARIS_TEST_INT", "42")
	t.Setenv("ARIS_TEST_BAD_INT", "bad")
	t.Setenv("ARIS_TEST_STRING", "value")
	t.Setenv("ARIS_TEST_BOOL", "true")
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
	if !EnvBool("ARIS_TEST_BOOL", false) {
		t.Fatal("expected parsed bool")
	}
	if !EnvBool("ARIS_TEST_BAD_BOOL", true) || !EnvBool("ARIS_TEST_MISSING_BOOL", true) {
		t.Fatal("expected bool fallback")
	}
}
