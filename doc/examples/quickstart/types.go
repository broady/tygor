// Package quickstart provides simple example types for documentation.
package quickstart

import "time"

// [snippet:types]
type News struct {
	ID        int32      `json:"id"`
	Title     string     `json:"title"`
	Body      *string    `json:"body"`
	CreatedAt *time.Time `json:"created_at"`
}

type ListNewsParams struct {
	Limit  *int32 `json:"limit"`
	Offset *int32 `json:"offset"`
}

type CreateNewsParams struct {
	Title string  `json:"title" validate:"required,min=3"`
	Body  *string `json:"body"`
}

// [/snippet:types]

// Ensure time is used (for compilation).
var _ = time.Time{}
