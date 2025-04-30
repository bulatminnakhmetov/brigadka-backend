package logging

import (
	"bufio"
	"bytes"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// errorResponseWriter is a custom response writer that captures both the status code and response body
type errorResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
	body       *bytes.Buffer // Buffer to store response body for error logging
}

// WriteHeader overrides the original WriteHeader to capture the status code
func (erw *errorResponseWriter) WriteHeader(code int) {
	erw.statusCode = code
	erw.written = true
	erw.ResponseWriter.WriteHeader(code)
}

// Write overrides the Write method to capture the response body and set status code to 200 if not previously set
func (erw *errorResponseWriter) Write(b []byte) (int, error) {
	if !erw.written {
		erw.statusCode = http.StatusOK
		erw.written = true
	}

	// If this is an error response, store the body in our buffer
	if erw.statusCode < 200 || erw.statusCode >= 300 {
		erw.body.Write(b)
	}

	return erw.ResponseWriter.Write(b)
}

// Hijack implements the http.Hijacker interface to support WebSockets
func (erw *errorResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := erw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// ErrorLogger returns a middleware that logs all non-2xx responses with their error messages
func ErrorLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer that captures the status code and body
		erw := &errorResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default status code
			body:           &bytes.Buffer{},
		}

		// Call the next handler
		next.ServeHTTP(erw, r)

		// If the status code is not 2xx (error), log the request details and error message
		if erw.statusCode < 200 || erw.statusCode >= 300 {
			duration := time.Since(start)

			// Extract the error message from the response body
			errorMessage := strings.TrimSpace(erw.body.String())
			if len(errorMessage) > 200 {
				// Truncate long error messages
				errorMessage = errorMessage[:200] + "..."
			}

			// Create a more detailed log entry
			log.Printf(
				"ERROR [%d] %s %s %s - User-Agent: %s - Message: %s",
				erw.statusCode,
				r.Method,
				r.URL.Path,
				duration,
				r.UserAgent(),
				errorMessage,
			)

			// Optional: Log request headers for more context
			// log.Printf("Request Headers: %v", r.Header)

			// Optional: Log client IP
			clientIP := r.Header.Get("X-Forwarded-For")
			if clientIP == "" {
				clientIP = r.RemoteAddr
			}
			log.Printf("Client IP: %s", clientIP)
		}
	})
}
