package api

import "time"

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
