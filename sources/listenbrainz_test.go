package sources

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func responseWithHeaders(headers map[string]string) *http.Response {
	resp := &http.Response{Header: make(http.Header)}
	for name, value := range headers {
		resp.Header.Set(name, value)
	}
	return resp
}

func TestNextRequestDelay(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected time.Duration
	}{
		{
			name:     "no rate limit headers",
			headers:  nil,
			expected: 1 * time.Second,
		},
		{
			name:     "plenty of requests remaining",
			headers:  map[string]string{"X-RateLimit-Remaining": "50"},
			expected: 1 * time.Second,
		},
		{
			name:     "low remaining waits for window reset",
			headers:  map[string]string{"X-RateLimit-Remaining": "1", "X-RateLimit-Reset-In": "30"},
			expected: 31 * time.Second,
		},
		{
			name:     "low remaining without reset time",
			headers:  map[string]string{"X-RateLimit-Remaining": "0"},
			expected: 1 * time.Second,
		},
		{
			name:     "unparseable headers",
			headers:  map[string]string{"X-RateLimit-Remaining": "soon", "X-RateLimit-Reset-In": "later"},
			expected: 1 * time.Second,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, nextRequestDelay(responseWithHeaders(test.headers)))
		})
	}
}

func TestRetryDelay(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		fallback time.Duration
		expected time.Duration
	}{
		{
			name:     "reset time given by server",
			headers:  map[string]string{"X-RateLimit-Reset-In": "20"},
			fallback: 10 * time.Second,
			expected: 25 * time.Second,
		},
		{
			name:     "no headers uses fallback",
			headers:  nil,
			fallback: 40 * time.Second,
			expected: 40 * time.Second,
		},
		{
			name:     "unparseable reset time uses fallback",
			headers:  map[string]string{"X-RateLimit-Reset-In": "later"},
			fallback: 10 * time.Second,
			expected: 10 * time.Second,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, retryDelay(responseWithHeaders(test.headers), test.fallback))
		})
	}
}
