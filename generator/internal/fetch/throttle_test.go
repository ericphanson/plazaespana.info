package fetch

import (
	"testing"
	"time"
)

func TestRequestThrottle_FirstRequest(t *testing.T) {
	throttle := NewRequestThrottle(100 * time.Millisecond)

	start := time.Now()
	delay, err := throttle.Wait("https://example.com/api")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
	if delay != 0 {
		t.Errorf("First request delay = %v, want 0", delay)
	}
	if elapsed > 10*time.Millisecond {
		t.Errorf("First request took %v, expected no delay", elapsed)
	}
}

func TestRequestThrottle_SubsequentRequest(t *testing.T) {
	throttle := NewRequestThrottle(100 * time.Millisecond)

	// First request
	_, err := throttle.Wait("https://example.com/api")
	if err != nil {
		t.Fatalf("First Wait failed: %v", err)
	}

	// Second request immediately after
	start := time.Now()
	delay, err := throttle.Wait("https://example.com/api")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Second Wait failed: %v", err)
	}

	// Should have delayed ~100ms
	if delay < 90*time.Millisecond || delay > 110*time.Millisecond {
		t.Errorf("Delay = %v, want ~100ms", delay)
	}
	if elapsed < 90*time.Millisecond {
		t.Errorf("Actual elapsed = %v, expected ~100ms delay", elapsed)
	}
}

func TestRequestThrottle_DifferentHosts(t *testing.T) {
	throttle := NewRequestThrottle(100 * time.Millisecond)

	// First request to example.com
	_, err := throttle.Wait("https://example.com/api")
	if err != nil {
		t.Fatalf("Wait for example.com failed: %v", err)
	}

	// Immediate request to different.com (should not be throttled)
	start := time.Now()
	delay, err := throttle.Wait("https://different.com/api")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Wait for different.com failed: %v", err)
	}

	if delay != 0 {
		t.Errorf("Different host delay = %v, want 0", delay)
	}
	if elapsed > 10*time.Millisecond {
		t.Errorf("Different host took %v, expected no delay", elapsed)
	}
}

func TestRequestThrottle_DelayExpired(t *testing.T) {
	throttle := NewRequestThrottle(50 * time.Millisecond)

	// First request
	_, err := throttle.Wait("https://example.com/api")
	if err != nil {
		t.Fatalf("First Wait failed: %v", err)
	}

	// Wait longer than minDelay
	time.Sleep(60 * time.Millisecond)

	// Second request (should not delay)
	start := time.Now()
	delay, err := throttle.Wait("https://example.com/api")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Second Wait failed: %v", err)
	}

	if delay != 0 {
		t.Errorf("Delay after expiration = %v, want 0", delay)
	}
	if elapsed > 10*time.Millisecond {
		t.Errorf("Request took %v, expected no delay", elapsed)
	}
}

func TestRequestThrottle_InvalidURL(t *testing.T) {
	throttle := NewRequestThrottle(100 * time.Millisecond)

	_, err := throttle.Wait("://invalid-url")
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}
