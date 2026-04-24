// Package apierrors — JSON API için standart hata yapısı.
// RFC 7807 (Problem Details for HTTP APIs) tabanlı.
package apierrors

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
)

// APIError — tüm JSON API hataları bu yapıyla döner.
type APIError struct {
	Status  int         `json:"status"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (e *APIError) Error() string { return e.Message }

// Hazır hata constructor'ları
func BadRequest(message string, details ...interface{}) *APIError {
	e := &APIError{Status: http.StatusBadRequest, Code: "BAD_REQUEST", Message: message}
	if len(details) > 0 {
		e.Details = details[0]
	}
	return e
}

func Unauthorized(message string) *APIError {
	return &APIError{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: message}
}

func Forbidden(message string) *APIError {
	return &APIError{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: message}
}

func NotFound(resource string) *APIError {
	return &APIError{
		Status:  http.StatusNotFound,
		Code:    "NOT_FOUND",
		Message: resource + " bulunamadı",
	}
}

func Conflict(message string) *APIError {
	return &APIError{Status: http.StatusConflict, Code: "CONFLICT", Message: message}
}

func UnprocessableEntity(message string, details interface{}) *APIError {
	return &APIError{
		Status:  http.StatusUnprocessableEntity,
		Code:    "VALIDATION_ERROR",
		Message: message,
		Details: details,
	}
}

func TooManyRequests() *APIError {
	return &APIError{
		Status:  http.StatusTooManyRequests,
		Code:    "RATE_LIMITED",
		Message: "Çok fazla istek. Lütfen bekleyin.",
	}
}

func Internal(message string) *APIError {
	return &APIError{
		Status:  http.StatusInternalServerError,
		Code:    "INTERNAL_ERROR",
		Message: message,
	}
}

// Send — APIError'ı JSON olarak yanıtlar.
func Send(c fiber.Ctx, err *APIError) error {
	return c.Status(err.Status).JSON(fiber.Map{
		"error": err,
	})
}

// APIResponse — başarılı API yanıtı için standart kapsayıcı.
type APIResponse struct {
	Data    interface{} `json:"data"`
	Meta    interface{} `json:"meta,omitempty"`
	Message string      `json:"message,omitempty"`
}

// OK — 200 başarılı yanıt.
func OK(c fiber.Ctx, data interface{}, meta ...interface{}) error {
	resp := APIResponse{Data: data}
	if len(meta) > 0 {
		resp.Meta = meta[0]
	}
	return c.Status(http.StatusOK).JSON(resp)
}

// Created — 201 oluşturuldu yanıtı.
func Created(c fiber.Ctx, data interface{}) error {
	return c.Status(http.StatusCreated).JSON(APIResponse{Data: data})
}

// NoContent — 204 içerik yok yanıtı.
func NoContent(c fiber.Ctx) error {
	return c.SendStatus(http.StatusNoContent)
}
