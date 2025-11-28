// Package api contains types that showcase different TypeScript generation options.
package api

import "time"

// Status represents the state of a task.
type Status string

const (
	// StatusPending indicates the task is waiting to be started.
	StatusPending Status = "pending"
	// StatusInProgress indicates the task is currently being worked on.
	StatusInProgress Status = "in_progress"
	// StatusCompleted indicates the task has been finished.
	StatusCompleted Status = "completed"
	// StatusCancelled indicates the task was cancelled.
	StatusCancelled Status = "cancelled"
)

// Priority represents task urgency levels.
type Priority int

const (
	// PriorityLow is for non-urgent tasks.
	PriorityLow Priority = 1
	// PriorityMedium is the default priority.
	PriorityMedium Priority = 2
	// PriorityHigh is for urgent tasks.
	PriorityHigh Priority = 3
	// PriorityCritical is for immediate attention.
	PriorityCritical Priority = 4
)

// Task represents a work item in the system.
type Task struct {
	// ID is the unique identifier for the task.
	ID string `json:"id"`
	// Title is the task's headline.
	Title string `json:"title"`
	// Description is the detailed task description.
	Description *string `json:"description,omitempty"`
	// Status is the current state of the task.
	Status Status `json:"status"`
	// Priority determines the task's urgency.
	Priority Priority `json:"priority"`
	// Assignee is the user responsible for the task.
	Assignee *string `json:"assignee,omitempty"`
	// DueDate is when the task should be completed.
	DueDate *time.Time `json:"due_date,omitempty"`
	// Tags are labels for categorization.
	Tags []string `json:"tags"`
	// CreatedAt is when the task was created.
	CreatedAt time.Time `json:"created_at"`
}

// CreateTaskRequest contains the data needed to create a new task.
type CreateTaskRequest struct {
	Title       string   `json:"title" validate:"required,min=3"`
	Description *string  `json:"description,omitempty"`
	Priority    Priority `json:"priority"`
	Assignee    *string  `json:"assignee,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// ListTasksParams contains query parameters for listing tasks.
type ListTasksParams struct {
	// Status filters tasks by their current status.
	Status *Status `json:"status,omitempty" schema:"status"`
	// Assignee filters tasks by assigned user.
	Assignee *string `json:"assignee,omitempty" schema:"assignee"`
	// Limit is the maximum number of results.
	Limit *int `json:"limit,omitempty" schema:"limit"`
	// Offset is the pagination offset.
	Offset *int `json:"offset,omitempty" schema:"offset"`
}
