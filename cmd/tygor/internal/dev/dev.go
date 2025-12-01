package dev

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/broady/tygor"
)

//go:embed testdata/rawrdata
var rawrData string

type Cmd struct {
	RPCDir string `help:"Directory containing generated RPC files." default:"./src/rpc" name:"rpc-dir"`
	Port   int    `help:"Port to listen on." default:"9000" short:"p"`
	Watch  bool   `help:"Watch for file changes." short:"w"`
}

func (c *Cmd) Run() error {
	svc := &Service{
		rpcDir: c.RPCDir,
	}
	app := NewApp(svc)

	// Mount under /__tygor to avoid conflicts with user services
	mux := http.NewServeMux()
	mux.Handle("/__tygor/", http.StripPrefix("/__tygor", app.Handler()))

	addr := fmt.Sprintf("localhost:%d", c.Port)
	fmt.Printf("tygor dev listening on http://%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

// NewApp creates a tygor App with the devtools service registered.
// Used both for the runtime server and for type generation.
func NewApp(svc *Service) *tygor.App {
	app := tygor.NewApp()
	devtools := app.Service("Devtools")
	devtools.Register("GetDiscovery", tygor.Query(svc.GetDiscovery))
	devtools.Register("GetSource", tygor.Query(svc.GetSource))
	devtools.Register("GetStatus", tygor.Query(svc.GetStatus))
	devtools.Register("UpdateStatus", tygor.Exec(svc.UpdateStatus))
	devtools.Register("Reload", tygor.Exec(svc.Reload))
	return app
}

// SetupApp returns a tygor.App for type generation.
// This is discovered by `tygor gen` via signature detection.
func SetupApp() *tygor.App {
	return NewApp(&Service{})
}

// Service implements the devtools server endpoints.
type Service struct {
	rpcDir    string
	appStatus *AppStatus // status reported by vite plugin
}

// GetDiscoveryRequest is the request for Devtools.GetDiscovery.
type GetDiscoveryRequest struct{}

// GetDiscoveryResponse returns the discovery.json content.
type GetDiscoveryResponse struct {
	// Discovery is the raw discovery.json content as a JSON object.
	Discovery json.RawMessage `json:"discovery"`
}

// GetDiscovery returns the discovery.json from the rpc-dir.
func (s *Service) GetDiscovery(ctx context.Context, req *GetDiscoveryRequest) (*GetDiscoveryResponse, error) {
	path := filepath.Join(s.rpcDir, "discovery.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, tygor.NewError(tygor.CodeNotFound, "discovery.json not found - run tygor gen first")
		}
		return nil, err
	}
	return &GetDiscoveryResponse{Discovery: data}, nil
}

// GetSourceRequest is the request for Devtools.GetSource.
type GetSourceRequest struct {
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Context int    `json:"context,omitempty"` // lines of context around Line (default 5)
}

// SourceLine represents a single line of source code.
type SourceLine struct {
	Num       int    `json:"num"`
	Content   string `json:"content"`
	Highlight bool   `json:"highlight,omitempty"`
}

// GetSourceResponse returns source code with context.
type GetSourceResponse struct {
	File     string       `json:"file"`
	Language string       `json:"language"`
	Lines    []SourceLine `json:"lines"`
	Context  int          `json:"context"`
}

// GetSource returns source code with context around a line.
func (s *Service) GetSource(ctx context.Context, req *GetSourceRequest) (*GetSourceResponse, error) {
	// TODO: implement source reading with security checks
	return nil, tygor.NewError(tygor.CodeNotImplemented, "not yet implemented")
}

// AppStatus represents the status of the user's app as reported by vite plugin.
type AppStatus struct {
	Status string `json:"status"` // "running", "building", "error", "starting"
	Port   int    `json:"port,omitempty"`
	Error  string `json:"error,omitempty"`
	Phase  string `json:"phase,omitempty"` // "prebuild", "build", "runtime"
}

// GetStatusRequest is the request for Devtools.GetStatus.
type GetStatusRequest struct {
	// Initial should be true on first request to receive one-time data.
	Initial bool `json:"initial,omitempty"`
}

// GetStatusResponse returns the combined status in flat format for the devtools UI.
// Status is a discriminated union: "ok", "error", "reloading", "starting", "disconnected".
type GetStatusResponse struct {
	Status   string   `json:"status"`
	Port     int      `json:"port,omitempty"`
	Error    string   `json:"error,omitempty"`
	Phase    string   `json:"phase,omitempty"` // "prebuild", "build", "runtime"
	Command  *string  `json:"command"`         // null when not applicable
	Cwd      string   `json:"cwd,omitempty"`
	ExitCode *int     `json:"exitCode"`           // null when not applicable
	RawrData []string `json:"rawrData,omitempty"` // sent on initial request
}

// GetStatus returns status for devtools UI consumption.
func (s *Service) GetStatus(ctx context.Context, req *GetStatusRequest) (*GetStatusResponse, error) {
	resp := &GetStatusResponse{}

	if s.appStatus == nil {
		resp.Status = "disconnected"
	} else {
		switch s.appStatus.Status {
		case "running":
			resp.Status = "ok"
			resp.Port = s.appStatus.Port
		case "building":
			resp.Status = "reloading"
		case "error":
			resp.Status = "error"
			resp.Error = s.appStatus.Error
			resp.Phase = s.appStatus.Phase
		default:
			resp.Status = "starting"
		}
	}

	if req.Initial {
		resp.RawrData = loadRawrData()
	}
	return resp, nil
}

// UpdateStatusRequest is sent by vite plugin to update app status.
type UpdateStatusRequest struct {
	App AppStatus `json:"app"`
}

// UpdateStatusResponse is the response for UpdateStatus.
type UpdateStatusResponse struct{}

// UpdateStatus updates the app status (called by vite plugin).
func (s *Service) UpdateStatus(ctx context.Context, req *UpdateStatusRequest) (*UpdateStatusResponse, error) {
	s.appStatus = &req.App
	return &UpdateStatusResponse{}, nil
}

// ReloadRequest triggers a reload of discovery.json.
type ReloadRequest struct {
	Files  []string `json:"files,omitempty"`
	Reason string   `json:"reason,omitempty"` // "file_change", "manual"
}

// ReloadResponse is the response for Reload.
type ReloadResponse struct{}

// Reload triggers reloading of discovery.json (called by vite plugin on file changes).
func (s *Service) Reload(ctx context.Context, req *ReloadRequest) (*ReloadResponse, error) {
	// For now, discovery is read on each request, so no action needed.
	// Future: could notify connected clients via SSE.
	return &ReloadResponse{}, nil
}

func loadRawrData() []string {
	var data []string
	for _, line := range strings.Split(rawrData, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			data = append(data, line)
		}
	}
	return data
}
