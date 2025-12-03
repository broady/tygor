package main

import (
	"context"
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/broady/tygor"
	"github.com/broady/tygor/tygorgen"

	"github.com/broady/tygor/examples/with-sqlc/sqlc"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

var db *sqlc.Queries

type Version struct {
	Value int `json:"value"`
}

// Version counter - bump on any mutation, clients refetch when it changes
var versionAtom = tygor.NewAtom(&Version{Value: 0})

// [snippet:app-setup]

func SetupApp() *tygor.App {
	app := tygor.NewApp()
	tasks := app.Service("Tasks")

	// Query endpoints
	tasks.Register("List", tygor.Query(ListTasks))
	tasks.Register("Get", tygor.Query(GetTask))
	tasks.Register("ListIncomplete", tygor.Query(ListIncomplete))

	// Mutation endpoints - bump version after success
	bumpVersion := func(ctx tygor.Context, req any, next tygor.HandlerFunc) (any, error) {
		res, err := next(ctx, req)
		if err == nil {
			versionAtom.Update(func(v *Version) *Version {
				return &Version{Value: v.Value + 1}
			})
		}
		return res, err
	}
	tasks.Register("Create", tygor.Exec(CreateTask).WithUnaryInterceptor(bumpVersion))
	tasks.Register("Update", tygor.Exec(db.UpdateTask).WithUnaryInterceptor(bumpVersion))
	tasks.Register("Delete", tygor.Exec(DeleteTask).WithUnaryInterceptor(bumpVersion))

	// Version stream - clients subscribe and refetch when it changes
	tasks.Register("Version", versionAtom.Handler())

	return app
}
// [/snippet:app-setup]

func TygorConfig(g *tygorgen.Generator) *tygorgen.Generator {
	return g.
		EnumStyle("union").
		OptionalType("undefined").
		WithDiscovery().
		WithFlavor(tygorgen.FlavorZod)
}

// [snippet:wrappers]

// Wrappers for sqlc methods with non-struct or int64 params

type ListTasksParams struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func ListTasks(ctx context.Context, p ListTasksParams) ([]sqlc.Task, error) {
	return db.ListTasks(ctx, sqlc.ListTasksParams{Limit: int64(p.Limit), Offset: int64(p.Offset)})
}

type GetTaskParams struct {
	ID int `json:"id"`
}

func GetTask(ctx context.Context, p GetTaskParams) (sqlc.Task, error) {
	return db.GetTask(ctx, p.ID)
}

type DeleteTaskParams struct {
	ID int `json:"id"`
}

func DeleteTask(ctx context.Context, p DeleteTaskParams) (tygor.Empty, error) {
	return nil, db.DeleteTask(ctx, p.ID)
}

func ListIncomplete(ctx context.Context, _ tygor.Empty) ([]sqlc.Task, error) {
	return db.ListIncompleteTasks(ctx)
}

type CreateTaskParams struct {
	Title       string  `json:"title" validate:"min=3"`
	Description *string `json:"description"`
}

func CreateTask(ctx context.Context, p CreateTaskParams) (sqlc.Task, error) {
	return db.CreateTask(ctx, sqlc.CreateTaskParams{
		Title:       p.Title,
		Description: p.Description,
	})
}
// [/snippet:wrappers]

func main() {
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	if p := os.Getenv("PORT"); p != "" {
		*port = p
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "data.sqlite.db"
	}

	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()

	if _, err := sqlDB.Exec(schema); err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	db = sqlc.New(sqlDB)
	app := SetupApp()

	addr := ":" + *port
	fmt.Printf("Server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, app.Handler()); err != nil {
		log.Fatal(err)
	}
}
