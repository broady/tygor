// Package v1 defines the v1 API types.
package v1

// [snippet:v1-types]

// User represents a user in v1 API.
type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// [/snippet:v1-types]

// GetUserRequest is the request for getting a v1 user.
type GetUserRequest struct {
	ID int64 `json:"id"`
}
