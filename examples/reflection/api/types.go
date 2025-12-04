package api

// [snippet:generic-types]

// PagedResponse wraps paginated data with metadata.
// This generic type demonstrates reflection provider's ability to handle
// instantiated generics (e.g., PagedResponse[User], PagedResponse[Post]).
type PagedResponse[T any] struct {
	// Data contains the page of results
	Data []T `json:"data"`
	// Total is the total number of items across all pages
	Total int `json:"total"`
	// Page is the current page number (1-indexed)
	Page int `json:"page"`
	// PageSize is the number of items per page
	PageSize int `json:"page_size"`
	// HasMore indicates if there are more pages available
	HasMore bool `json:"has_more"`
}

// Result wraps operation results with success/error status.
// This pattern is common when working with types from external packages
// where source analysis isn't available.
type Result[T any] struct {
	// Success indicates if the operation succeeded
	Success bool `json:"success"`
	// Data contains the result if successful
	Data *T `json:"data,omitempty"`
	// Error contains the error message if failed
	Error *string `json:"error,omitempty"`
}

// [/snippet:generic-types]

// [snippet:concrete-types]

// User represents a user in the system.
type User struct {
	ID       int32  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// Post represents a blog post.
type Post struct {
	ID       int32  `json:"id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	AuthorID int32  `json:"author_id"`
}

// [/snippet:concrete-types]

// [snippet:request-types]

// ListUsersParams contains pagination parameters for listing users.
type ListUsersParams struct {
	Page     int    `json:"page" schema:"page"`
	PageSize int    `json:"page_size" schema:"page_size"`
	Role     string `json:"role,omitempty" schema:"role"`
}

// GetUserParams contains parameters for fetching a single user.
type GetUserParams struct {
	ID int32 `json:"id" schema:"id"`
}

// CreatePostParams contains parameters for creating a post.
type CreatePostParams struct {
	Title    string `json:"title" validate:"required,min=3"`
	Content  string `json:"content" validate:"required,min=10"`
	AuthorID int32  `json:"author_id" validate:"required"`
}

// [/snippet:request-types]
