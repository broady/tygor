package api

//go:generate go run ../ -gen

import "time"

// NewsStatus represents the publication status of a news article.
type NewsStatus string

const (
	// NewsStatusDraft indicates the article is not yet published.
	NewsStatusDraft NewsStatus = "draft"
	// NewsStatusPublished indicates the article is publicly visible.
	NewsStatusPublished NewsStatus = "published"
	// NewsStatusArchived indicates the article has been archived.
	NewsStatusArchived NewsStatus = "archived"
)

// News represents a news article in the system.
type News struct {
	// ID is the unique identifier for the article.
	ID int32 `json:"id"`
	// Title is the article headline.
	Title string `json:"title"`
	// Body is the optional article content.
	Body *string `json:"body,omitempty"`
	// Status is the current publication status of the article.
	Status NewsStatus `json:"status"`
	// CreatedAt is the timestamp when the article was created.
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// ListNewsParams contains pagination parameters for listing news articles.
type ListNewsParams struct {
	// Limit is the maximum number of articles to return.
	Limit *int32 `json:"limit,omitempty"`
	// Offset is the number of articles to skip.
	Offset *int32 `json:"offset,omitempty"`
}

// CreateNewsParams contains the parameters for creating a new news article.
type CreateNewsParams struct {
	// Title is the article headline (required, 3-100 characters).
	Title string `json:"title" validate:"required,min=3"`
	// Body is the optional article content.
	Body *string `json:"body,omitempty"`
}
