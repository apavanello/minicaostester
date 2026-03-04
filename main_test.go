package main

import (
	"net/http"
	"os"
	"testing"
	"time"
)

func TestParseEnvInt(t *testing.T) {
	// Setup
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	// Test case 1: valid value
	if val := parseEnvInt("TEST_INT", 10); val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}

	// Test case 2: empty value (should return default)
	if val := parseEnvInt("NON_EXISTENT", 10); val != 10 {
		t.Errorf("Expected default 10, got %d", val)
	}

	// Test case 3: invalid value (should return default)
	os.Setenv("TEST_INVALID_INT", "abc")
	defer os.Unsetenv("TEST_INVALID_INT")
	if val := parseEnvInt("TEST_INVALID_INT", 10); val != 10 {
		t.Errorf("Expected default 10 for invalid input, got %d", val)
	}
}

func TestParseEnvBool(t *testing.T) {
	os.Setenv("TEST_BOOL_TRUE", "true")
	os.Setenv("TEST_BOOL_FALSE", "false")
	os.Setenv("TEST_BOOL_INVALID", "notabool")
	defer os.Unsetenv("TEST_BOOL_TRUE")
	defer os.Unsetenv("TEST_BOOL_FALSE")
	defer os.Unsetenv("TEST_BOOL_INVALID")

	if val := parseEnvBool("TEST_BOOL_TRUE", false); !val {
		t.Errorf("Expected true, got false")
	}

	if val := parseEnvBool("TEST_BOOL_FALSE", true); val {
		t.Errorf("Expected false, got true")
	}

	if val := parseEnvBool("TEST_BOOL_INVALID", true); !val {
		t.Errorf("Expected default true, got false")
	}

	if val := parseEnvBool("NON_EXISTENT", false); val {
		t.Errorf("Expected default false, got true")
	}
}

func TestBurnCPU(t *testing.T) {
	// We just want to ensure it runs and returns a duration.
	// We use a small complexity to avoid slowing down the test suite.
	dur := burnCPU(100)
	if dur < 0 {
		t.Errorf("Expected duration >= 0, got %v", dur)
	}
}

func TestComputeDelay(t *testing.T) {
	// Test normal range
	dur := computeDelay(10, 50)
	if dur < 10*time.Millisecond || dur >= 50*time.Millisecond {
		t.Errorf("Expected delay between 10ms and 50ms, got %v", dur)
	}

	// Test min >= max
	dur2 := computeDelay(50, 10)
	if dur2 != 50*time.Millisecond {
		t.Errorf("Expected delay to be exactly min (50ms) when min >= max, got %v", dur2)
	}
}

func TestPrioritizeStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		statuses []int
		expected int
	}{
		{"All 200s", []int{200, 200, 200}, 200},
		{"Single 500 overrides 200", []int{200, 500, 200}, 500},
		{"Single 400 overrides 200", []int{200, 404, 200}, 404},
		{"5xx overrides 4xx", []int{200, 404, 503}, 503},
		{"Highest 5xx wins", []int{500, 504, 502}, 504},
		{"Highest 4xx wins when no 5xx", []int{400, 404, 401}, 404},
		{"Empty list", []int{}, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := prioritizeStatusCode(tt.statuses); got != tt.expected {
				t.Errorf("prioritizeStatusCode(%v) = %v, want %v", tt.statuses, got, tt.expected)
			}
		})
	}
}

func TestGetDepth(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)

	if d := getDepth(req); d != 0 {
		t.Errorf("Expected default depth 0, got %d", d)
	}

	req.Header.Set("X-Call-Depth", "3")
	if d := getDepth(req); d != 3 {
		t.Errorf("Expected depth 3, got %d", d)
	}

	req.Header.Set("X-Call-Depth", "invalid")
	if d := getDepth(req); d != 0 {
		t.Errorf("Expected depth 0 for invalid header, got %d", d)
	}
}
