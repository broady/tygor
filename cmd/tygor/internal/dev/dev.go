package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/broady/tygor"
)

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

	addr := fmt.Sprintf("localhost:%d", c.Port)
	fmt.Printf("tygor dev listening on http://%s\n", addr)
	return http.ListenAndServe(addr, app.Handler())
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
type GetStatusRequest struct{}

// GetStatusResponse returns the combined devtools status.
type GetStatusResponse struct {
	Devtools DevtoolsStatus `json:"devtools"`
	App      *AppStatus     `json:"app,omitempty"`
}

// DevtoolsStatus is the status of the devtools server itself.
type DevtoolsStatus struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// GetStatus returns combined devtools and app status.
func (s *Service) GetStatus(ctx context.Context, req *GetStatusRequest) (*GetStatusResponse, error) {
	return &GetStatusResponse{
		Devtools: DevtoolsStatus{
			Status:  "ok",
			Version: "0.1.0", // TODO: get from version package
		},
		App: s.appStatus,
	}, nil
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
