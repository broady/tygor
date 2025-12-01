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

// Task list as an Atom - subscribers get current list and updates
var tasksAtom = tygor.NewAtom([]*api.Task{})
var nextID int32 = 1
var serverPort string

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
func CurrentTime(ctx context.Context, req tygor.Empty, emit *tygor.Emitter[*api.TimeUpdate]) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// Send immediately
	if err := emit.Send(&api.TimeUpdate{Time: time.Now()}); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case t := <-ticker.C:
			if err := emit.Send(&api.TimeUpdate{Time: t}); err != nil {
				return err
			}
		}
	}
}

func main() {
	portFlag := flag.String("port", "8080", "Server port")
	genFlag := flag.Bool("gen", false, "Generate TypeScript types")
	outDir := flag.String("out", "./src/rpc", "Output directory")
	flag.Parse()

	serverPort = *portFlag
	if p := os.Getenv("PORT"); p != "" {
		serverPort = p
	}

	// Simulate slow startup for testing blue/green deployment
	if delay := os.Getenv("STARTUP_DELAY"); delay != "" {
		d, err := time.ParseDuration(delay)
		if err == nil {
			fmt.Printf(" Simulating slow startup: %s\n", delay)
			time.Sleep(d)
		}
	}

	// No CORS needed - Vite proxies API requests in dev, same-origin in prod
	app := tygor.NewApp()

	// Note: Devtools functionality is now provided by `tygor dev` - no need to
	// register anything in the user's app! Discovery is served by tygor dev.

	system := app.Service("System")
	system.Register("Kill", tygor.Exec(Kill))

	tasksvc := app.Service("Tasks")
	tasksvc.Register("List", tygor.Query(ListTasks))
	tasksvc.Register("Subscribe", tasksAtom.Handler())
	tasksvc.Register("Time", tygor.StreamEmit(CurrentTime))
	tasksvc.Register("Create", tygor.Exec(CreateTask))
	tasksvc.Register("Toggle", tygor.Exec(ToggleTask))
	tasksvc.Register("MakeError", tygor.Exec(MakeError))

	if *genFlag {
		fmt.Printf("Generating types to %s...\n", *outDir)
		if err := os.MkdirAll(*outDir, 0755); err != nil {
			log.Fatal(err)
		}
		_, err := tygorgen.FromApp(app).
			EnumStyle("union").
			OptionalType("undefined").
			WithDiscovery().
			ToDir(*outDir)
		if err != nil {
			log.Fatalf("Generation failed: %v", err)
		}
		fmt.Println("Done.")
		return
	}

	addr := ":" + serverPort
	fmt.Printf("Server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, app.Handler()); err != nil {
		log.Fatal(err)
	}
}
