package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/broady/tygor/internal/discover"
)

func TestExec(t *testing.T) {
	t.Setenv("GOWORK", "off")

	dir := t.TempDir()

	tygorRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	goMod := `module testapp

go 1.21

require github.com/broady/tygor v0.7.4

replace github.com/broady/tygor => ` + tygorRoot + `
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Write main.go with unexported setupApp and main()
	mainGo := `package main

import (
	"context"
	"github.com/broady/tygor"
)

type HelloRequest struct {
	Name string
}

type HelloResponse struct {
	Message string
}

func setupApp() *tygor.App {
	app := tygor.NewApp()
	svc := app.Service("Test")
	svc.Register("Hello", tygor.Query(func(_ context.Context, req HelloRequest) (HelloResponse, error) {
		return HelloResponse{Message: "hello " + req.Name}, nil
	}))
	return app
}

func main() {
	setupApp().Handler()
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Copy go.sum from tygor root to get transitive dependency checksums
	goSum, err := os.ReadFile(filepath.Join(tygorRoot, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), goSum, 0644); err != nil {
		t.Fatal(err)
	}

	// Use go mod download to fetch dependencies without modifying go.sum
	cmd := exec.Command("go", "mod", "download")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod download: %v\n%s", err, out)
	}

	outDir := filepath.Join(dir, "rpc")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Run with unexported function name
	output, err := Exec(Options{
		Export: discover.Export{
			Name: "setupApp",
			Type: discover.ExportTypeApp,
		},
		OutDir: outDir,
		PkgDir: dir,
	})
	if err != nil {
		t.Fatalf("Exec() error: %v\nOutput: %s", err, output)
	}

	// Verify output was generated
	typesPath := filepath.Join(outDir, "types.ts")
	if _, err := os.Stat(typesPath); os.IsNotExist(err) {
		t.Errorf("types.ts was not generated")
	}

	manifestPath := filepath.Join(outDir, "manifest.ts")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Errorf("manifest.ts was not generated")
	}

	// Verify types_main.ts contains our types (types from package main go here)
	typesMainPath := filepath.Join(outDir, "types_main.ts")
	content, err := os.ReadFile(typesMainPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "HelloRequest") {
		t.Errorf("types_main.ts missing HelloRequest")
	}
	if !strings.Contains(string(content), "HelloResponse") {
		t.Errorf("types_main.ts missing HelloResponse")
	}
}

func TestRemoveMain(t *testing.T) {
	dir := t.TempDir()

	// Write a file with main()
	src := `package main

import "fmt"

func helper() {
	fmt.Println("helper")
}

func main() {
	helper()
}
`
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	hasMain, modified, err := removeMain(file)
	if err != nil {
		t.Fatal(err)
	}

	if !hasMain {
		t.Error("expected hasMain=true")
	}

	// Check that main is removed but helper remains
	modStr := string(modified)
	if strings.Contains(modStr, "func main()") {
		t.Error("main() was not removed")
	}
	if !strings.Contains(modStr, "func helper()") {
		t.Error("helper() was removed")
	}
}

func TestRemoveMainNoMain(t *testing.T) {
	dir := t.TempDir()

	// Write a file without main()
	src := `package main

func helper() {}
`
	file := filepath.Join(dir, "helper.go")
	if err := os.WriteFile(file, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	hasMain, modified, err := removeMain(file)
	if err != nil {
		t.Fatal(err)
	}

	if hasMain {
		t.Error("expected hasMain=false")
	}
	if modified != nil {
		t.Error("expected nil modified")
	}
}
