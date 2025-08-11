package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	server := &GardiyanServer{}
	
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.healthCheck)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Health check handler yanlış status code döndü: got %v want %v",
			status, http.StatusOK)
	}

	expected := "Gardiyan çalışıyor"
	if rr.Body.String() != expected {
		t.Errorf("Health check handler yanlış body döndü: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		filename    string
		expectedType string
	}{
		{"test.html", "text/html"},
		{"test.css", "text/css"},
		{"test.js", "application/javascript"},
		{"test.json", "application/json"},
		{"test.png", "image/png"},
		{"test.jpg", "image/jpeg"},
		{"test.gif", "image/gif"},
		{"test.pdf", "application/pdf"},
		{"test.txt", "text/plain"},
		{"test.unknown", "application/octet-stream"},
	}

	for _, test := range tests {
		result := getContentType(test.filename)
		if result != test.expectedType {
			t.Errorf("getContentType(%s) = %s; want %s", 
				test.filename, result, test.expectedType)
		}
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	// Test environment variable var
	os.Setenv("TEST_VAR", "test_value")
	result := getEnvOrDefault("TEST_VAR", "default")
	if result != "test_value" {
		t.Errorf("getEnvOrDefault() = %s; want test_value", result)
	}

	// Test environment variable yok
	os.Unsetenv("TEST_VAR")
	result = getEnvOrDefault("TEST_VAR", "default")
	if result != "default" {
		t.Errorf("getEnvOrDefault() = %s; want default", result)
	}
}
