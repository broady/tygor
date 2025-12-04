package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/broady/tygor"
	"github.com/broady/tygor/tygorgen"

	"github.com/broady/tygor/examples/devtools/api"
)

// SetupApp returns the tygor app for type generation.
func SetupApp() *tygor.App {
	app := tygor.NewApp()

	system := app.Service("System")
	system.Register("Kill", tygor.Exec(Kill))

	tasksvc := app.Service("Tasks")
	tasksvc.Register("List", tygor.Query(ListTasks))
	tasksvc.Register("SyncedList", tasksAtom.Handler())
	tasksvc.Register("Time", tygor.Stream(CurrentTime))
	tasksvc.Register("Create", tygor.Exec(CreateTask))
	tasksvc.Register("Toggle", tygor.Exec(ToggleTask))
	tasksvc.Register("MakeError", tygor.Exec(MakeError))

	return app
}

// TygorConfig configures TypeScript generation options.
func TygorConfig(g *tygorgen.Generator) *tygorgen.Generator {
	return g.
		EnumStyle("union").
		OptionalType("undefined").
		WithDiscovery()
}

// Task list as an Atom - subscribers get current list and updates
var tasksAtom = tygor.NewAtom([]*api.Task{})
var nextID int32 = 1

func Kill(ctx context.Context, req tygor.Empty) (tygor.Empty, error) {
	fmt.Println("Kill requested, shutting down...")
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()
	return nil, nil
}

func ListTasks(ctx context.Context, req *api.ListTasksParams) ([]*api.Task, error) {
	tasks := tasksAtom.Get()
	if req.ShowDone != nil && !*req.ShowDone {
		filtered := []*api.Task{}
		for _, t := range tasks {
			if !t.Done {
				filtered = append(filtered, t)
			}
		}
		return filtered, nil
	}
	if tasks == nil {
		return []*api.Task{}, nil
	}
	return tasks, nil
}

func CreateTask(ctx context.Context, req *api.CreateTaskParams) (*api.Task, error) {
	task := &api.Task{
		ID:    nextID,
		Title: req.Title,
		Done:  false,
	}
	nextID++

	// Update atom - this broadcasts to all subscribers
	tasksAtom.Update(func(tasks []*api.Task) []*api.Task {
		return append(tasks, task)
	})
	return task, nil
}

func MakeError(ctx context.Context, req tygor.Empty) (tygor.Empty, error) {
	return nil, errors.New("hi from tygor!")
}

func ToggleTask(ctx context.Context, req *api.ToggleTaskParams) (*api.Task, error) {
	var found *api.Task
	tasksAtom.Update(func(tasks []*api.Task) []*api.Task {
		for _, t := range tasks {
			if t.ID == req.ID {
				t.Done = !t.Done
				found = t
				break
			}
		}
		return tasks
	})
	if found == nil {
		return nil, tygor.NewError(tygor.CodeNotFound, "task not found")
	}
	return found, nil
}

// CurrentTime streams the current time every second
func CurrentTime(_ context.Context, req tygor.Empty, stream tygor.StreamWriter[*api.TimeUpdate]) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// Send immediately
	if err := stream.Send(&api.TimeUpdate{Time: time.Now()}); err != nil {
		return err
	}

	for t := range ticker.C {
		if err := stream.Send(&api.TimeUpdate{Time: t}); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	if p := os.Getenv("PORT"); p != "" {
		*port = p
	}

	// Simulate slow startup for testing blue/green deployment
	if delay := os.Getenv("STARTUP_DELAY"); delay != "" {
		d, err := time.ParseDuration(delay)
		if err == nil {
			fmt.Printf("Simulating slow startup: %s\n", delay)
			time.Sleep(d)
		}
	}

	app := SetupApp()

	addr := ":" + *port
	fmt.Printf("Server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, app.Handler()); err != nil {
		log.Fatal(err)
	}
}
