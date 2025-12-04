package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/broady/tygor"
	"github.com/broady/tygor/tygorgen"

	"github.com/broady/tygor/examples/react/api"
)

// In-memory task store
var (
	tasks   = []*api.Task{}
	tasksMu sync.Mutex
	nextID  int32 = 1
)

func GetRuntimeInfo(ctx context.Context, req tygor.Empty) (*api.RuntimeInfo, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return &api.RuntimeInfo{
		Version:       runtime.Version(),
		NumGoroutines: runtime.NumGoroutine(),
		NumCPU:        runtime.NumCPU(),
		Memory: api.MemoryStats{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			NumGC:      m.NumGC,
		},
	}, nil
}

func StreamRuntimeInfo(ctx context.Context, req tygor.Empty, stream tygor.StreamWriter[*api.RuntimeInfo]) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		info, err := GetRuntimeInfo(ctx, nil)
		if err != nil {
			return err
		}
		if err := stream.Send(info); err != nil {
			return err
		}
	}
	return nil
}

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

// SetupApp configures the tygor application.
// This export is used by `tygor gen` for type generation.
func SetupApp() *tygor.App {
	app := tygor.NewApp()

	system := app.Service("System")
	system.Register("Info", tygor.Query(GetRuntimeInfo))
	system.Register("InfoStream", tygor.Stream(StreamRuntimeInfo))

	tasksSvc := app.Service("Tasks")
	tasksSvc.Register("List", tygor.Query(ListTasks))
	tasksSvc.Register("Create", tygor.Exec(CreateTask))
	tasksSvc.Register("Toggle", tygor.Exec(ToggleTask))

	return app
}

// TygorConfig configures the TypeScript generator.
func TygorConfig(g *tygorgen.Generator) *tygorgen.Generator {
	return g.
		EnumStyle("union").
		OptionalType("undefined").
		WithDiscovery().
		WithFlavor(tygorgen.FlavorZod)
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
