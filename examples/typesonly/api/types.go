// Package api contains type definitions for standalone TypeScript generation.
package api

import "time"

// User represents a user in the system.
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  Role   `json:"role"`
	// Profile is nil for users who haven't completed onboarding.
	Profile   *Profile       `json:"profile,omitempty"`
	Tags      []string       `json:"tags"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"createdAt"`
}

// Profile contains optional user profile information.
type Profile struct {
	Bio    string            `json:"bio"`
	Avatar *string           `json:"avatar,omitempty"`
	Links  map[string]string `json:"links"`
}

// Role represents the access level of a user.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// Page is a generic paginated response.
type Page[T any] struct {
	Items   []T  `json:"items"`
	Total   int  `json:"total"`
	HasMore bool `json:"hasMore"`
}
