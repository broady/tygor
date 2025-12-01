package api

//go:generate go run ../ -gen -out ../src/rpc

// Task represents a todo item.
type Task struct {
	// ID is the unique identifier.
	ID int32 `json:"id"`
	// Title is the task description.
	Title string `json:"title"`
	// Done indicates if the task is completed.
	Done bool `json:"done"`
}

// ListTasksParams contains parameters for listing tasks.
type ListTasksParams struct {
	// ShowDone filters to show completed tasks.
	ShowDone *bool `json:"show_done,omitempty" schema:"show_done"`
}

// CreateTaskParams contains parameters for creating a task.
type CreateTaskParams struct {
	Title string `json:"title" validate:"required,min=1"`
}

// ToggleTaskParams identifies which task to toggle.
type ToggleTaskParams struct {
	ID int32 `json:"id" validate:"required"`
}
