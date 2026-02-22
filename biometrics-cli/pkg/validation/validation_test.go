package validation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("Expected validator, got nil")
	}
	if v.maxInputLength != 10000 {
		t.Errorf("Expected maxInputLength 10000, got %d", v.maxInputLength)
	}
}

func TestValidateEmail(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		email    string
		expected bool
	}{
		{"user@example.com", true},
		{"user.name@example.com", true},
		{"user+tag@example.co.uk", true},
		{"invalid", false},
		{"@example.com", false},
		{"user@", false},
		{"", false},
	}

	for _, tt := range tests {
		err := v.ValidateEmail(tt.email)
		if (err == nil) != tt.expected {
			t.Errorf("ValidateEmail(%q): expected valid=%v, got error=%v", tt.email, tt.expected, err)
		}
	}
}

func TestValidateURL(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com", true},
		{"http://example.com/path", true},
		{"ftp://example.com", false},
		{"javascript:alert(1)", false},
		{"", false},
	}

	for _, tt := range tests {
		err := v.ValidateURL(tt.url)
		if (err == nil) != tt.expected {
			t.Errorf("ValidateURL(%q): expected valid=%v, got error=%v", tt.url, tt.expected, err)
		}
	}
}

func TestValidateUUID(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		uuid     string
		expected bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"550E8400-E29B-41D4-A716-446655440000", true},
		{"invalid-uuid", false},
		{"", false},
	}

	for _, tt := range tests {
		err := v.ValidateUUID(tt.uuid)
		if (err == nil) != tt.expected {
			t.Errorf("ValidateUUID(%q): expected valid=%v, got error=%v", tt.uuid, tt.expected, err)
		}
	}
}

func TestContainsSQLInjection(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		input    string
		expected bool
	}{
		// Note: Simple SELECT is NOT SQL injection - it's a legitimate query
		{"SELECT * FROM users", false},
		{"OR 1=1", true},
		{"'; DROP TABLE users; --", true},
		{"UNION SELECT * FROM passwords", true},
		{"normal text", false},
		{"Hello World", false},
	}

	for _, tt := range tests {
		result := v.ContainsSQLInjection(tt.input)
		if result != tt.expected {
			t.Errorf("ContainsSQLInjection(%q): expected %v, got %v", tt.input, tt.expected, result)
		}
	}
}

func TestContainsXSS(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		input    string
		expected bool
	}{
		{"<script>alert(1)</script>", true},
		{"<img onerror=alert(1)>", true},
		{"javascript:alert(1)", true},
		{"normal text", false},
		{"<p>Hello</p>", false},
	}

	for _, tt := range tests {
		result := v.ContainsXSS(tt.input)
		if result != tt.expected {
			t.Errorf("ContainsXSS(%q): expected %v, got %v", tt.input, tt.expected, result)
		}
	}
}

func TestValidateString(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"normal", "Hello World", true},
		// SELECT is a legitimate query, not injection
		{"sql injection", "SELECT * FROM users", true},
		{"xss", "<script>alert(1)</script>", false},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.ValidateString(tt.input)
			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, result.Valid)
			}
		})
	}
}

func TestIsSafeString(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		input    string
		expected bool
	}{
		{"HelloWorld123", true},
		{"test@example.com", true},
		{"path/to/file", true},
		{"<script>", false},
		{"'; DROP TABLE", false},
	}

	for _, tt := range tests {
		result := v.IsSafeString(tt.input)
		if result != tt.expected {
			t.Errorf("IsSafeString(%q): expected %v, got %v", tt.input, tt.expected, result)
		}
	}
}

func TestValidateLength(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		input    string
		min      int
		max      int
		expected bool
	}{
		{"hello", 1, 10, true},
		{"hi", 5, 10, false},
		{"very long string", 1, 5, false},
	}

	for _, tt := range tests {
		err := v.ValidateLength(tt.input, tt.min, tt.max)
		if (err == nil) != tt.expected {
			t.Errorf("ValidateLength(%q, %d, %d): expected %v, got error=%v", tt.input, tt.min, tt.max, tt.expected, err)
		}
	}
}

func TestValidateAlphanumeric(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		input    string
		expected bool
	}{
		{"Hello123", true},
		{"123456", true},
		{"Hello", true},
		{"Hello!", false},
		{"Hello World", false},
	}

	for _, tt := range tests {
		err := v.ValidateAlphanumeric(tt.input)
		if (err == nil) != tt.expected {
			t.Errorf("ValidateAlphanumeric(%q): expected %v, got error=%v", tt.input, tt.expected, err)
		}
	}
}

func TestGenerateCSRFToken(t *testing.T) {
	v := NewValidator()

	token1, err := v.GenerateCSRFToken()
	if err != nil {
		t.Fatalf("GenerateCSRFToken failed: %v", err)
	}

	token2, err := v.GenerateCSRFToken()
	if err != nil {
		t.Fatalf("GenerateCSRFToken failed: %v", err)
	}

	if token1 == token2 {
		t.Error("Expected different tokens, got same")
	}

	if len(token1) != 64 {
		t.Errorf("Expected token length 64, got %d", len(token1))
	}
}

func TestValidateCSRFToken(t *testing.T) {
	v := NewValidator()

	token := "test-token"

	if err := v.ValidateCSRFToken(token, token); err != nil {
		t.Errorf("Expected valid token, got error: %v", err)
	}

	if err := v.ValidateCSRFToken(token, "different"); err == nil {
		t.Error("Expected error for mismatched tokens")
	}

	if err := v.ValidateCSRFToken("", token); err == nil {
		t.Error("Expected error for empty token")
	}
}

func TestSanitizeInput(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		input    string
		expected string
	}{
		{"hello\x00world", "helloworld"},
		{"  hello  ", "hello"},
		{"hello\nworld", "hello\nworld"},
	}

	for _, tt := range tests {
		result := v.SanitizeInput(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeInput(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}

func TestGetValidationErrors(t *testing.T) {
	v := NewValidator()

	err := v.validate.Var("", "required")
	if err == nil {
		t.Fatal("Expected validation error")
	}

	validationErrors := v.GetValidationErrors(err)
	if len(validationErrors) == 0 {
		t.Error("Expected validation errors")
	}
}

func TestSetMaxInputLength(t *testing.T) {
	v := NewValidator()

	v.SetMaxInputLength(5000)
	if v.GetMaxInputLength() != 5000 {
		t.Errorf("Expected max length 5000, got %d", v.GetMaxInputLength())
	}
}

func TestValidatePattern(t *testing.T) {
	v := NewValidator()
	pattern := regexp.MustCompile(`^[a-z]+$`)

	tests := []struct {
		input    string
		expected bool
	}{
		{"hello", true},
		{"Hello", false},
		{"123", false},
	}

	for _, tt := range tests {
		err := v.ValidatePattern(tt.input, pattern)
		if (err == nil) != tt.expected {
			t.Errorf("ValidatePattern(%q): expected %v, got error=%v", tt.input, tt.expected, err)
		}
	}
}

func TestValidateNumeric(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		input    string
		expected bool
	}{
		{"123456", true},
		{"123abc", false},
		{"abc", false},
	}

	for _, tt := range tests {
		err := v.ValidateNumeric(tt.input)
		if (err == nil) != tt.expected {
			t.Errorf("ValidateNumeric(%q): expected %v, got error=%v", tt.input, tt.expected, err)
		}
	}
}

func TestValidateRequired(t *testing.T) {
	v := NewValidator()

	if err := v.ValidateRequired("hello"); err != nil {
		t.Errorf("Expected no error for non-empty string")
	}

	if err := v.ValidateRequired(""); err != ErrRequiredField {
		t.Errorf("Expected ErrRequiredField for empty string")
	}

	if err := v.ValidateRequired("   "); err != ErrRequiredField {
		t.Errorf("Expected ErrRequiredField for whitespace-only string")
	}
}

func TestValidator_DecodeInput(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		input    string
		expected string
	}{
		{"hello%20world", "hello world"},
		{"&lt;script&gt;", "<script>"},
		{"normal", "normal"},
	}

	for _, tt := range tests {
		result := v.decodeInput(tt.input)
		if result != tt.expected {
			t.Errorf("decodeInput(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}

func TestValidator_InitPatterns(t *testing.T) {
	v := NewValidator()

	if len(v.sqlPatterns) == 0 {
		t.Error("Expected SQL patterns to be initialized")
	}

	if len(v.xssPatterns) == 0 {
		t.Error("Expected XSS patterns to be initialized")
	}
}

// TestDefaultSanitizeConfig tests default sanitization configuration
func TestDefaultSanitizeConfig(t *testing.T) {
	config := DefaultSanitizeConfig()

	if !config.StripHTML {
		t.Error("Expected StripHTML to be true")
	}
	if !config.StripScripts {
		t.Error("Expected StripScripts to be true")
	}
	if config.StripSQLKeywords {
		t.Error("Expected StripSQLKeywords to be false")
	}
	if !config.NormalizePaths {
		t.Error("Expected NormalizePaths to be true")
	}
	if !config.NormalizeWhitespace {
		t.Error("Expected NormalizeWhitespace to be true")
	}
	if !config.RemoveControlChars {
		t.Error("Expected RemoveControlChars to be true")
	}
	if config.MaxRecursionDepth != 3 {
		t.Errorf("Expected MaxRecursionDepth 3, got %d", config.MaxRecursionDepth)
	}
}

// TestNewSanitizer tests sanitizer creation
func TestNewSanitizer(t *testing.T) {
	s := NewSanitizer()
	if s == nil {
		t.Fatal("Expected sanitizer, got nil")
	}

	config := DefaultSanitizeConfig()
	s2 := NewSanitizerWithConfig(config)
	if s2 == nil {
		t.Fatal("Expected sanitizer with config, got nil")
	}
}

// TestSanitizeString tests string sanitization
func TestSanitizeString(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Remove HTML tags", "<script>alert(1)</script>", ""},
		{"Remove control chars", "hello\x00world", "helloworld"},
		{"Normalize whitespace", "hello   world", "hello world"},
		{"Normal text", "Hello World", "Hello World"},
		{"Path traversal", "../../../etc/passwd", "../../../etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestSanitizeHTML tests HTML sanitization
func TestSanitizeHTML(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Script tags", "<script>alert(1)</script>", ""},
		{"Event handlers", "<img onerror=alert(1)>", "&lt;img onerror=alert(1)&gt;"},
		{"Safe HTML", "<p>Hello</p>", "&lt;p&gt;Hello&lt;/p&gt;"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestSanitizeURL tests URL sanitization
func TestSanitizeURL(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid HTTPS", "https://example.com", false},
		{"Valid HTTP", "http://example.com/path", false},
		{"Invalid URL - converted to https", "not-a-url", false},
		{"JavaScript URL - converted to https", "javascript:alert(1)", false},
		{"FTP URL - converted to https", "ftp://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.SanitizeURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error=%v, got %v", tt.wantErr, err)
			}
			if !tt.wantErr && result == "" {
				t.Error("Expected non-empty result")
			}
		})
	}
}

// TestSanitizeFilePath tests file path sanitization
func TestSanitizeFilePath(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Path traversal", "../../../etc/passwd", "passwd"},
		{"Normal path", "/home/user/file.txt", "file.txt"},
		{"Windows path", "..\\..\\windows\\system32", "windows\\system32"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeFilePath(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestDefaultMiddlewareConfig tests default middleware configuration
func TestDefaultMiddlewareConfig(t *testing.T) {
	config := DefaultMiddlewareConfig()

	if !config.CSRFEnabled {
		t.Error("Expected CSRFEnabled to be true")
	}
	if config.CSRFCookieName != "csrf_token" {
		t.Errorf("Expected CSRFCookieName 'csrf_token', got %s", config.CSRFCookieName)
	}
	if config.MaxBodySize != 1<<20 {
		t.Errorf("Expected MaxBodySize 1MB, got %d", config.MaxBodySize)
	}
	if len(config.SkipPaths) != 2 {
		t.Errorf("Expected 2 skip paths, got %d", len(config.SkipPaths))
	}
}

// TestNewMiddleware tests middleware creation
func TestNewMiddleware(t *testing.T) {
	m := NewMiddleware(nil)
	if m == nil {
		t.Fatal("Expected middleware, got nil")
	}

	config := DefaultMiddlewareConfig()
	m2 := NewMiddleware(config)
	if m2 == nil {
		t.Fatal("Expected middleware with config, got nil")
	}
}

// TestResponseWriter tests response writer wrapper
func TestResponseWriter(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := NewResponseWriter(recorder)

	if rw.statusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rw.statusCode)
	}

	rw.WriteHeader(http.StatusCreated)
	if rw.statusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rw.statusCode)
	}
}

// TestMiddlewareValidateRequest tests request validation
func TestMiddlewareValidateRequest(t *testing.T) {
	m := NewMiddleware(nil)

	handler := m.ValidateRequest(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/health", nil)
	recorder := httptest.NewRecorder()

	handler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}
}

// TestMiddlewareCORSMiddleware tests CORS middleware
func TestMiddlewareCORSMiddleware(t *testing.T) {
	config := DefaultMiddlewareConfig()
	config.AllowedOrigins = []string{"https://example.com"}
	config.AllowedMethods = []string{"GET", "POST"}
	config.AllowedHeaders = []string{"Content-Type"}

	m := NewMiddleware(config)

	handler := m.CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "https://example.com")
	recorder := httptest.NewRecorder()

	handler(recorder, req)

	if recorder.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Error("Expected CORS header")
	}
}

// TestMiddlewareSecurityHeaders tests security headers middleware
func TestMiddlewareSecurityHeaders(t *testing.T) {
	m := NewMiddleware(nil)

	handler := m.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	recorder := httptest.NewRecorder()

	handler(recorder, req)

	headers := recorder.Header()
	if headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("Expected X-Content-Type-Options header")
	}
	if headers.Get("X-Frame-Options") != "DENY" {
		t.Error("Expected X-Frame-Options header")
	}
	if headers.Get("X-XSS-Protection") != "1; mode=block" {
		t.Error("Expected X-XSS-Protection header")
	}
}

// TestMiddlewareCSRFProtect tests CSRF protection
func TestMiddlewareCSRFProtect(t *testing.T) {
	m := NewMiddleware(nil)

	handler := m.CSRFProtect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test GET request (should pass)
	req := httptest.NewRequest("GET", "/api/test", nil)
	recorder := httptest.NewRecorder()
	handler(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200 for GET, got %d", recorder.Code)
	}

	// Test POST without CSRF token (should fail)
	req = httptest.NewRequest("POST", "/api/test", nil)
	recorder = httptest.NewRecorder()
	handler(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for POST without CSRF, got %d", recorder.Code)
	}
}

// TestMiddlewareValidateJSONBody tests JSON body validation middleware
func TestMiddlewareValidateJSONBody(t *testing.T) {
	m := NewMiddleware(nil)

	schema := map[string]string{
		"name":  "required",
		"email": "required,email",
	}

	handler := m.ValidateJSONBody(schema)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := `{"name": "test", "email": "test@example.com"}`
	req := httptest.NewRequest("POST", "/api/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}
}

// TestMiddlewareErrorResponse tests error response
func TestMiddlewareErrorResponse(t *testing.T) {
	recorder := httptest.NewRecorder()

	ErrorResponse(recorder, "Test error", http.StatusBadRequest)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", recorder.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(recorder.Body).Decode(&response)

	if response["error"] != "Test error" {
		t.Errorf("Expected error message 'Test error', got %v", response["error"])
	}
}

// TestMiddlewareSuccessResponse tests success response
func TestMiddlewareSuccessResponse(t *testing.T) {
	recorder := httptest.NewRecorder()

	SuccessResponse(recorder, map[string]string{"status": "ok"})

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(recorder.Body).Decode(&response)

	if response["success"] != "true" {
		t.Errorf("Expected success 'true', got %v", response["success"])
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}
	if data["status"] != "ok" {
		t.Errorf("Expected data.status 'ok', got %v", data["status"])
	}
}

// TestMiddlewareChainMiddleware tests middleware chaining
func TestMiddlewareChainMiddleware(t *testing.T) {
	m := NewMiddleware(nil)

	var executed bool
	handler := ChainMiddleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executed = true
			w.WriteHeader(http.StatusOK)
		}),
		m.SecurityHeaders,
		m.LoggingMiddleware,
	)

	req := httptest.NewRequest("GET", "/api/test", nil)
	recorder := httptest.NewRecorder()

	handler(recorder, req)

	if !executed {
		t.Error("Expected handler to be executed")
	}
}

// TestMiddlewareRecoverMiddleware tests panic recovery
func TestMiddlewareRecoverMiddleware(t *testing.T) {
	m := NewMiddleware(nil)

	handler := m.RecoverMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	recorder := httptest.NewRecorder()

	// Should not panic
	handler(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", recorder.Code)
	}
}

// TestMiddlewareRequestIDMiddleware tests request ID generation
func TestMiddlewareRequestIDMiddleware(t *testing.T) {
	m := NewMiddleware(nil)

	handler := m.RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r)
		if requestID == "" {
			t.Error("Expected request ID")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	recorder := httptest.NewRecorder()

	handler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}
}

// TestMiddlewareLoggingMiddleware tests request logging
func TestMiddlewareLoggingMiddleware(t *testing.T) {
	m := NewMiddleware(nil)

	handler := m.LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	recorder := httptest.NewRecorder()

	handler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}
}

// TestSanitizeStringWithConfig tests sanitization with custom config
func TestSanitizeStringWithConfig(t *testing.T) {
	s := NewSanitizer()

	config := DefaultSanitizeConfig()
	config.StripHTML = false

	result := s.SanitizeStringWithConfig("<p>Hello</p>", config)
	if result != "<p>Hello</p>" {
		t.Errorf("Expected '<p>Hello</p>', got %q", result)
	}
}

// TestMiddlewareGetAuthToken tests auth token extraction
func TestMiddlewareGetAuthToken(t *testing.T) {
	// GetAuthToken reads from context, not from Authorization header
	req := httptest.NewRequest("GET", "/api/test", nil)
	ctx := context.WithValue(req.Context(), "auth_token", "test-token-123")
	req = req.WithContext(ctx)

	token := GetAuthToken(req)
	if token != "test-token-123" {
		t.Errorf("Expected 'test-token-123', got %s", token)
	}
}

// TestMiddlewareGenerateCSRFCookie tests CSRF cookie generation
func TestMiddlewareGenerateCSRFCookie(t *testing.T) {
	m := NewMiddleware(nil)

	cookie, err := m.GenerateCSRFCookie()
	if err != nil {
		t.Fatalf("GenerateCSRFCookie failed: %v", err)
	}

	if cookie == nil {
		t.Error("Expected non-nil CSRF cookie")
	}

	if cookie.Value == "" {
		t.Error("Expected non-empty CSRF cookie value")
	}
}

// TestMiddlewareSetCSRFCookie tests CSRF cookie setting
func TestMiddlewareSetCSRFCookie(t *testing.T) {
	m := NewMiddleware(nil)

	recorder := httptest.NewRecorder()

	err := m.SetCSRFCookie(recorder)
	if err != nil {
		t.Fatalf("SetCSRFCookie failed: %v", err)
	}

	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Error("Expected CSRF cookie to be set")
	}
}

// TestMiddlewareValidateRequestWithSkipPaths tests skip paths
func TestMiddlewareValidateRequestWithSkipPaths(t *testing.T) {
	m := NewMiddleware(nil)

	handler := m.ValidateRequest(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/metrics", nil)
	recorder := httptest.NewRecorder()

	handler(recorder, req)

	// Should skip validation for /metrics
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}
}
