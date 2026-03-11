package errors

import (
	"errors"
	"strings"
)

type Class string

const (
	ClassRateLimit           Class = "rate_limit"
	ClassQuota               Class = "quota"
	ClassTimeout             Class = "timeout"
	ClassTransient           Class = "transient"
	ClassProviderUnavailable Class = "provider_unavailable"
	ClassFatal               Class = "fatal"
)

type ProviderError struct {
	Class   Class
	Message string
	Cause   error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return "provider error"
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func IsRecoverable(class Class) bool {
	switch class {
	case ClassRateLimit, ClassQuota, ClassTimeout, ClassTransient, ClassProviderUnavailable:
		return true
	default:
		return false
	}
}

func Classify(err error) (Class, string) {
	if err == nil {
		return "", ""
	}
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		class := providerErr.Class
		if class == "" {
			class = ClassFatal
		}
		return class, providerErr.Error()
	}

	text := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(text, "rate limit") || strings.Contains(text, "too many requests"):
		return ClassRateLimit, err.Error()
	case strings.Contains(text, "quota"):
		return ClassQuota, err.Error()
	case strings.Contains(text, "timeout") || strings.Contains(text, "deadline exceeded"):
		return ClassTimeout, err.Error()
	case strings.Contains(text, "unavailable") || strings.Contains(text, "not found") || strings.Contains(text, "not ready"):
		return ClassProviderUnavailable, err.Error()
	default:
		return ClassTransient, err.Error()
	}
}
