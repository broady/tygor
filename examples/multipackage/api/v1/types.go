// Package v1 defines the v1 API types.
package v1

// User represents a user in v1 API.
type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// GetUserRequest is the request for getting a v1 user.
type GetUserRequest struct {
	ID int64 `json:"id"`
}
