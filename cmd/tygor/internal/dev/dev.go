package dev

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

type Cmd struct {
	RPCDir string `help:"Directory containing generated RPC files." default:"./src/rpc" name:"rpc-dir"`
	Port   int    `help:"Port to listen on." default:"9000" short:"p"`
	Watch  bool   `help:"Watch for file changes." short:"w"`
}

func (c *Cmd) Run() error {
	mux := http.NewServeMux()

	// Discovery endpoint - serves discovery.json from rpc-dir
	discoveryPath := filepath.Join(c.RPCDir, "discovery.json")
	mux.HandleFunc("GET /__tygor/discovery", func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(discoveryPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "discovery.json not found - run tygor gen first", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	// TODO(3.3): Source endpoint
	// TODO(3.4): Status endpoint
	// TODO(3.5): Control API

	mux.HandleFunc("GET /__tygor/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	addr := fmt.Sprintf("localhost:%d", c.Port)
	fmt.Printf("tygor dev listening on http://%s\n", addr)
	return http.ListenAndServe(addr, mux)
}
