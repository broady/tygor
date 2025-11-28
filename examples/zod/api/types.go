package api

//go:generate go run ../ -gen -out ../client/src/rpc

import "time"

// [snippet:user-type]

// User demonstrates common validation patterns.
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username" validate:"required,min=3,max=20,alphanum"`
	Email     string    `json:"email" validate:"required,email"`
	Website   *string   `json:"website,omitempty" validate:"omitempty,url"`
	Age       *int32    `json:"age,omitempty" validate:"omitempty,gte=0,lte=150"`
	CreatedAt time.Time `json:"created_at"`
}

// [/snippet:user-type]

// [snippet:task-type]

// Task demonstrates nested types and oneof validation.
type Task struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title" validate:"required,min=1,max=200"`
	Description *string    `json:"description,omitempty" validate:"omitempty,max=2000"`
	Priority    string     `json:"priority" validate:"required,oneof=low medium high critical"`
	AssigneeID  *int64     `json:"assignee_id,omitempty" validate:"omitempty,gt=0"`
	Tags        []string   `json:"tags" validate:"max=10"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Completed   bool       `json:"completed"`
}

// [/snippet:task-type]

// [snippet:create-user-request]

// CreateUserRequest demonstrates request validation.
type CreateUserRequest struct {
	Username string  `json:"username" validate:"required,min=3,max=20,alphanum"`
	Email    string  `json:"email" validate:"required,email"`
	Password string  `json:"password" validate:"required,min=8,max=72"`
	Website  *string `json:"website,omitempty" validate:"omitempty,url"`
	Age      *int32  `json:"age,omitempty" validate:"omitempty,gte=13,lte=150"`
}

// [/snippet:create-user-request]

// [snippet:create-task-request]

// CreateTaskRequest demonstrates complex request validation.
type CreateTaskRequest struct {
	Title       string   `json:"title" validate:"required,min=1,max=200"`
	Description *string  `json:"description,omitempty" validate:"omitempty,max=2000"`
	Priority    string   `json:"priority" validate:"required,oneof=low medium high critical"`
	AssigneeID  *int64   `json:"assignee_id,omitempty" validate:"omitempty,gt=0"`
	Tags        []string `json:"tags" validate:"max=10"`
}

// [/snippet:create-task-request]

// [snippet:update-task-request]

// UpdateTaskRequest demonstrates partial update patterns.
type UpdateTaskRequest struct {
	TaskID      int64   `json:"task_id" validate:"required,gt=0"`
	Title       *string `json:"title,omitempty" validate:"omitempty,min=1,max=200"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=2000"`
	Priority    *string `json:"priority,omitempty" validate:"omitempty,oneof=low medium high critical"`
	AssigneeID  *int64  `json:"assignee_id,omitempty" validate:"omitempty,gt=0"`
	Completed   *bool   `json:"completed,omitempty"`
}

// [/snippet:update-task-request]

// ListParams contains pagination parameters.
type ListParams struct {
	Limit  int32 `json:"limit" schema:"limit" validate:"gte=1,lte=100"`
	Offset int32 `json:"offset" schema:"offset" validate:"gte=0"`
}

// GetTaskParams contains parameters for getting a task.
type GetTaskParams struct {
	TaskID int64 `json:"task_id" schema:"task_id" validate:"required,gt=0"`
}
