package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/broady/tygor"
	"github.com/broady/tygor/examples/react/api"
	"github.com/broady/tygor/middleware"
	"github.com/broady/tygor/tygorgen"
)

// In-memory task store
var (
	tasks   = []*api.Task{}
	tasksMu sync.Mutex
	nextID  int32 = 1
)

// [snippet:handlers collapse]

func ListTasks(ctx context.Context, req *api.ListTasksParams) ([]*api.Task, error) {
	tasksMu.Lock()
	defer tasksMu.Unlock()

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
	tasksMu.Lock()
	defer tasksMu.Unlock()

	task := &api.Task{
		ID:    nextID,
		Title: req.Title,
		Done:  false,
	}
	nextID++
	tasks = append(tasks, task)
	return task, nil
}

func ToggleTask(ctx context.Context, req *api.ToggleTaskParams) (*api.Task, error) {
	tasksMu.Lock()
	defer tasksMu.Unlock()

	for _, t := range tasks {
		if t.ID == req.ID {
			t.Done = !t.Done
			return t, nil
		}
	}
	return nil, tygor.NewError(tygor.CodeNotFound, "task not found")
}

// [/snippet:handlers]

func main() {
	port := flag.String("port", "8080", "Server port")
	genFlag := flag.Bool("gen", false, "Generate TypeScript types")
	outDir := flag.String("out", "./client/src/rpc", "Output directory")
	flag.Parse()

	// [snippet:app-setup]

	app := tygor.NewApp().
		WithMiddleware(middleware.CORS(middleware.CORSAllowAll)).
		WithUnaryInterceptor(middleware.LoggingInterceptor(nil))

	tasks := app.Service("Tasks")
	tasks.Register("List", tygor.Query(ListTasks))
	tasks.Register("Create", tygor.Exec(CreateTask))
	tasks.Register("Toggle", tygor.Exec(ToggleTask))

	// [/snippet:app-setup]

	if *genFlag {
		fmt.Printf("Generating types to %s...\n", *outDir)
		if err := os.MkdirAll(*outDir, 0755); err != nil {
			log.Fatal(err)
		}
		_, err := tygorgen.FromApp(app).
			EnumStyle("union").
			OptionalType("undefined").
			ToDir(*outDir)
		if err != nil {
			log.Fatalf("Generation failed: %v", err)
		}
		fmt.Println("Done.")
		return
	}

	addr := ":" + *port
	fmt.Printf("Server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, app.Handler()); err != nil {
		log.Fatal(err)
	}
}
