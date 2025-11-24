package api

import "time"

// User represents a user in the system.
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Post represents a blog post.
type Post struct {
	ID        int64     `json:"id"`
	AuthorID  int64     `json:"author_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Published bool      `json:"published"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Comment represents a comment on a post.
type Comment struct {
	ID        int64     `json:"id"`
	PostID    int64     `json:"post_id"`
	AuthorID  int64     `json:"author_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateUserRequest is the request to create a new user.
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=20,alphanum"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// LoginRequest is the request to login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse contains the authentication token.
type LoginResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

// CreatePostRequest is the request to create a new post.
type CreatePostRequest struct {
	Title   string `json:"title" validate:"required,min=5,max=200"`
	Content string `json:"content" validate:"required,min=10"`
}

// UpdatePostRequest is the request to update a post.
type UpdatePostRequest struct {
	PostID  int64   `json:"post_id" validate:"required,gt=0"`
	Title   *string `json:"title,omitempty" validate:"omitempty,min=5,max=200"`
	Content *string `json:"content,omitempty" validate:"omitempty,min=10"`
}

// ListPostsParams are the query parameters for listing posts.
type ListPostsParams struct {
	AuthorID  *int64 `schema:"author_id"`
	Published *bool  `schema:"published"`
	Limit     int32  `schema:"limit"`
	Offset    int32  `schema:"offset"`
}

// GetPostParams are the query parameters for getting a post.
type GetPostParams struct {
	PostID int64 `schema:"post_id" validate:"required,gt=0"`
}

// PublishPostRequest marks a post as published.
type PublishPostRequest struct {
	PostID int64 `json:"post_id" validate:"required,gt=0"`
}

// CreateCommentRequest is the request to create a comment.
type CreateCommentRequest struct {
	PostID  int64  `json:"post_id" validate:"required,gt=0"`
	Content string `json:"content" validate:"required,min=1,max=500"`
}

// ListCommentsParams are the query parameters for listing comments.
type ListCommentsParams struct {
	PostID int64 `schema:"post_id" validate:"required,gt=0"`
	Limit  int32 `schema:"limit"`
	Offset int32 `schema:"offset"`
}
