package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/broady/tygor"
	"github.com/broady/tygor/examples/zod/api"
	"github.com/broady/tygor/tygorgen"
)

// In-memory database (for demo purposes)
var (
	dbMu       sync.RWMutex
	users            = make(map[int64]*api.User)
	tasks            = make(map[int64]*api.Task)
	nextUserID int64 = 1
	nextTaskID int64 = 1
)

func init() {
	// Seed with demo data
	now := time.Now()

	users[1] = &api.User{
		ID:        1,
		Username:  "alice",
		Email:     "alice@example.com",
		CreatedAt: now,
	}
	nextUserID = 2

	tasks[1] = &api.Task{
		ID:        1,
		Title:     "Complete project documentation",
		Priority:  "medium",
		Tags:      []string{"docs", "important"},
		Completed: false,
	}
	nextTaskID = 2
}

// --- User Handlers ---

func CreateUser(ctx context.Context, req *api.CreateUserRequest) (*api.User, error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	for _, u := range users {
		if u.Email == req.Email {
			return nil, tygor.NewError(tygor.CodeInvalidArgument, "email already registered")
		}
		if u.Username == req.Username {
			return nil, tygor.NewError(tygor.CodeInvalidArgument, "username already taken")
		}
	}

	user := &api.User{
		ID:        nextUserID,
		Username:  req.Username,
		Email:     req.Email,
		Website:   req.Website,
		Age:       req.Age,
		CreatedAt: time.Now(),
	}
	users[nextUserID] = user
	nextUserID++

	return user, nil
}

func ListUsers(ctx context.Context, req *api.ListParams) ([]*api.User, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()

	result := make([]*api.User, 0, len(users))
	for _, u := range users {
		result = append(result, u)
	}

	// Apply pagination
	start := int(req.Offset)
	if start >= len(result) {
		return []*api.User{}, nil
	}

	end := start + int(req.Limit)
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

// --- Task Handlers ---

func CreateTask(ctx context.Context, req *api.CreateTaskRequest) (*api.Task, error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	task := &api.Task{
		ID:          nextTaskID,
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		AssigneeID:  req.AssigneeID,
		Tags:        req.Tags,
		Completed:   false,
	}
	tasks[nextTaskID] = task
	nextTaskID++

	return task, nil
}

func GetTask(ctx context.Context, req *api.GetTaskParams) (*api.Task, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()

	task, ok := tasks[req.TaskID]
	if !ok {
		return nil, tygor.NewError(tygor.CodeNotFound, "task not found")
	}

	return task, nil
}

func UpdateTask(ctx context.Context, req *api.UpdateTaskRequest) (*api.Task, error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	task, ok := tasks[req.TaskID]
	if !ok {
		return nil, tygor.NewError(tygor.CodeNotFound, "task not found")
	}

	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = req.Description
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.AssigneeID != nil {
		task.AssigneeID = req.AssigneeID
	}
	if req.Completed != nil {
		task.Completed = *req.Completed
	}

	return task, nil
}

func ListTasks(ctx context.Context, req *api.ListParams) ([]*api.Task, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()

	result := make([]*api.Task, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, t)
	}

	// Apply pagination
	start := int(req.Offset)
	if start >= len(result) {
		return []*api.Task{}, nil
	}

	end := start + int(req.Limit)
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

// SetupApp configures the tygor application.
// This export is used by `tygor gen` for type generation.
func SetupApp() *tygor.App {
	app := tygor.NewApp()

	// User Service
	userService := app.Service("Users")
	userService.Register("Create", tygor.Exec(CreateUser))
	userService.Register("List", tygor.Query(ListUsers))

	// Task Service
	taskService := app.Service("Tasks")
	taskService.Register("Create", tygor.Exec(CreateTask))
	taskService.Register("Get", tygor.Query(GetTask))
	taskService.Register("Update", tygor.Exec(UpdateTask))
	taskService.Register("List", tygor.Query(ListTasks))

	return app
}

// TygorConfig configures the TypeScript generator.
func TygorConfig(g *tygorgen.Generator) *tygorgen.Generator {
	// [snippet:zod-generation]
	return g.
		WithFlavor(tygorgen.FlavorZod).
		WithFlavor(tygorgen.FlavorZodMini)
	// [/snippet:zod-generation]
}

func main() {
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	if p := os.Getenv("PORT"); p != "" {
		*port = p
	}

	app := SetupApp()

	addr := ":" + *port
	fmt.Printf("Server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, app.Handler()); err != nil {
		log.Fatal(err)
	}
}
