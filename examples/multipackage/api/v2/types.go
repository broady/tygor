// Package v2 defines the v2 API types with expanded fields.
package v2

// User represents a user in v2 API with additional fields.
type User struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

// GetUserRequest is the request for getting a v2 user.
type GetUserRequest struct {
	ID int64 `json:"id"`
}
