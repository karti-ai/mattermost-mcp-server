package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ErrorCategory string

const (
	CategoryFile    ErrorCategory = "file"
	CategoryAuth    ErrorCategory = "auth"
	CategoryRepo    ErrorCategory = "repo"
	CategoryChannel ErrorCategory = "channel"
	CategoryUser    ErrorCategory = "user"
	CategoryMessage ErrorCategory = "message"
	CategoryNetwork ErrorCategory = "network"
	CategoryUnknown ErrorCategory = "unknown"
)

type EnhancedError struct {
	Original   error
	Translated string
	Category   ErrorCategory
	Operation  string
	Context    map[string]string
	Timestamp  time.Time
}

func (e *EnhancedError) Error() string {
	if e.Translated != "" {
		return e.Translated
	}
	if e.Original != nil {
		return e.Original.Error()
	}
	return "unknown error"
}

func (e *EnhancedError) Unwrap() error {
	return e.Original
}

func (e *EnhancedError) WithContext(key, value string) *EnhancedError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

func (e *EnhancedError) WithOperation(op string) *EnhancedError {
	e.Operation = op
	return e
}

func (e *EnhancedError) WithParam(key, value string) *EnhancedError {
	return e.WithContext(key, value)
}

func (e *EnhancedError) FormatDetailed() string {
	details := map[string]any{
		"error":     e.Error(),
		"category":  e.Category,
		"timestamp": e.Timestamp.Format(time.RFC3339),
	}

	if e.Operation != "" {
		details["operation"] = e.Operation
	}

	if len(e.Context) > 0 {
		details["context"] = e.Context
	}

	if e.Original != nil && e.Original.Error() != e.Error() {
		details["original"] = e.Original.Error()
	}

	jsonBytes, err := json.MarshalIndent(details, "", "  ")
	if err != nil {
		return e.Format()
	}

	return string(jsonBytes)
}

func (e *EnhancedError) Format() string {
	var parts []string

	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("Operation: %s", e.Operation))
	}

	parts = append(parts, fmt.Sprintf("Error: %s", e.Error()))

	if e.Category != "" && e.Category != CategoryUnknown {
		parts = append(parts, fmt.Sprintf("Category: %s", e.Category))
	}

	if len(e.Context) > 0 {
		var ctxParts []string
		for k, v := range e.Context {
			ctxParts = append(ctxParts, fmt.Sprintf("%s=%s", k, v))
		}
		parts = append(parts, fmt.Sprintf("Context: %s", strings.Join(ctxParts, ", ")))
	}

	if e.Original != nil && e.Original.Error() != e.Error() {
		parts = append(parts, fmt.Sprintf("Original: %s", e.Original.Error()))
	}

	return strings.Join(parts, " | ")
}

func TranslateError(err error, context map[string]string) error {
	if err == nil {
		return nil
	}

	var existing *EnhancedError
	if errors.As(err, &existing) {
		if context != nil {
			for k, v := range context {
				existing.WithContext(k, v)
			}
		}
		return existing
	}

	translated, category := translateErrorMessage(err)

	operation := ""
	if context != nil {
		operation = context["operation"]
	}

	enhanced := &EnhancedError{
		Original:   err,
		Translated: translated,
		Category:   category,
		Operation:  operation,
		Context:    context,
		Timestamp:  time.Now().UTC(),
	}

	return enhanced
}

func translateErrorMessage(err error) (string, ErrorCategory) {
	if err == nil {
		return "", CategoryUnknown
	}

	msg := err.Error()
	lowerMsg := strings.ToLower(msg)

	if strings.Contains(msg, "404") {
		return "Resource not found", CategoryUnknown
	}

	if strings.Contains(msg, "401") {
		return "Authentication failed - check your access token", CategoryAuth
	}

	if strings.Contains(msg, "403") {
		return "Permission denied - you don't have access to this resource", CategoryAuth
	}

	translations := []struct {
		pattern  string
		message  string
		category ErrorCategory
	}{
		{"GetUser", "User not found", CategoryUser},
		{"GetChannel", "Channel not found", CategoryChannel},
		{"GetPost", "Message not found", CategoryMessage},
		{"CreatePost", "Failed to create message", CategoryMessage},
		{"UpdatePost", "Failed to update message", CategoryMessage},
		{"DeletePost", "Failed to delete message", CategoryMessage},
		{"CreateChannel", "Failed to create channel", CategoryChannel},
	}

	for _, t := range translations {
		if strings.Contains(msg, t.pattern) {
			return t.message, t.category
		}
	}

	if strings.Contains(lowerMsg, "timeout") || strings.Contains(lowerMsg, "deadline exceeded") {
		return "Request timed out - the server took too long to respond", CategoryNetwork
	}

	if strings.Contains(lowerMsg, "connection refused") || strings.Contains(lowerMsg, "no such host") {
		return "Network error - cannot connect to server", CategoryNetwork
	}

	return msg, CategoryUnknown
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	var enhanced *EnhancedError
	if errors.As(err, &enhanced) {
		switch enhanced.Category {
		case CategoryFile, CategoryRepo, CategoryChannel, CategoryUser, CategoryMessage:
			return true
		}
		return strings.Contains(enhanced.Translated, "not found")
	}

	var httpErr interface{ Error() string }
	if errors.As(err, &httpErr) {
		if strings.Contains(httpErr.Error(), "404") {
			return true
		}
	}

	lowerMsg := strings.ToLower(err.Error())
	return strings.Contains(lowerMsg, "not found") || strings.Contains(lowerMsg, "404")
}

func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	var enhanced *EnhancedError
	if errors.As(err, &enhanced) {
		return enhanced.Category == CategoryAuth
	}

	msg := err.Error()
	if strings.Contains(msg, "401") || strings.Contains(msg, "403") {
		return true
	}

	lowerMsg := strings.ToLower(msg)
	return strings.Contains(lowerMsg, "authentication") ||
		strings.Contains(lowerMsg, "unauthorized") ||
		strings.Contains(lowerMsg, "permission denied") ||
		strings.Contains(lowerMsg, "forbidden")
}

func IsTimeout(err error) bool {
	if err == nil {
		return false
	}

	var enhanced *EnhancedError
	if errors.As(err, &enhanced) {
		return enhanced.Category == CategoryNetwork || strings.Contains(enhanced.Translated, "timed out")
	}

	lowerMsg := strings.ToLower(err.Error())
	return strings.Contains(lowerMsg, "timeout") ||
		strings.Contains(lowerMsg, "deadline exceeded") ||
		strings.Contains(lowerMsg, "context deadline")
}

func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	var enhanced *EnhancedError
	if errors.As(err, &enhanced) {
		return enhanced.Category == CategoryNetwork
	}

	lowerMsg := strings.ToLower(err.Error())
	return strings.Contains(lowerMsg, "connection") ||
		strings.Contains(lowerMsg, "network") ||
		strings.Contains(lowerMsg, "no such host") ||
		strings.Contains(lowerMsg, "dial tcp")
}

func NewEnhancedError(original error, translated string, category ErrorCategory) *EnhancedError {
	return &EnhancedError{
		Original:   original,
		Translated: translated,
		Category:   category,
		Context:    make(map[string]string),
		Timestamp:  time.Now().UTC(),
	}
}

func Wrap(err error, operation string) error {
	if err == nil {
		return nil
	}
	return TranslateError(err, map[string]string{"operation": operation})
}

type HTTPError interface {
	error
	Status() int
}

type statusError struct {
	status  int
	message string
}

func (e *statusError) Error() string { return e.message }
func (e *statusError) Status() int   { return e.status }

func IsHTTPError(err error, statusCode int) bool {
	if err == nil {
		return false
	}

	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Status() == statusCode
	}

	msg := err.Error()
	return strings.Contains(msg, fmt.Sprintf("status %d", statusCode)) ||
		strings.Contains(msg, fmt.Sprintf("%d", statusCode))
}

func IsUnauthorized(err error) bool {
	return IsHTTPError(err, http.StatusUnauthorized)
}

func IsForbidden(err error) bool {
	return IsHTTPError(err, http.StatusForbidden)
}

func IsNotFoundHTTP(err error) bool {
	return IsHTTPError(err, http.StatusNotFound)
}

func IsServerError(err error) bool {
	if err == nil {
		return false
	}

	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Status() >= 500 && httpErr.Status() < 600
	}

	msg := err.Error()
	for i := 500; i < 600; i++ {
		if strings.Contains(msg, fmt.Sprintf("status %d", i)) ||
			strings.Contains(msg, fmt.Sprintf("%d", i)) {
			return true
		}
	}
	return false
}
