// Package devtools provides a devtools service for the tygor vite plugin.
package devtools

import (
	"context"
	_ "embed"
	"runtime"
	"strings"

	"github.com/broady/tygor"
)

//go:embed testdata/rawrdata
var rawrData string

// Service provides devtools endpoints for the vite plugin.
// Register it on your App to enable devtools integration:
//
//	app := tygor.NewApp()
//	devtools.New(app, 8080).Register()
type Service struct {
	app  *tygor.App
	port int
}

// New creates a new devtools service.
func New(app *tygor.App, port int) *Service {
	return &Service{app: app, port: port}
}

// Register adds the devtools service to the app.
func (s *Service) Register() {
	svc := s.app.Service("Devtools")
	svc.Register("Ping", tygor.Query(s.Ping))
	svc.Register("Info", tygor.Query(s.Info))
	svc.Register("Status", tygor.Query(s.Status))
}

// PingRequest is the request for Devtools.Ping.
type PingRequest struct{}

// PingResponse is the response for Devtools.Ping.
type PingResponse struct {
	OK bool `json:"ok"`
}

// Ping is a simple health check endpoint for heartbeat.
func (s *Service) Ping(ctx context.Context, req *PingRequest) (*PingResponse, error) {
	return &PingResponse{OK: true}, nil
}

// InfoRequest is the request for Devtools.Info.
type InfoRequest struct{}

// InfoResponse provides runtime information about the server.
type InfoResponse struct {
	Port          int         `json:"port"`
	Version       string      `json:"version"`
	NumGoroutines int         `json:"num_goroutines"`
	NumCPU        int         `json:"num_cpu"`
	Memory        MemoryStats `json:"memory"`
}

// MemoryStats contains memory statistics.
type MemoryStats struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
}

// Info returns runtime information about the server.
func (s *Service) Info(ctx context.Context, req *InfoRequest) (*InfoResponse, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return &InfoResponse{
		Port:          s.port,
		Version:       runtime.Version(),
		NumGoroutines: runtime.NumGoroutine(),
		NumCPU:        runtime.NumCPU(),
		Memory: MemoryStats{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			NumGC:      m.NumGC,
		},
	}, nil
}

// StatusRequest is the request for Devtools.Status.
type StatusRequest struct {
	// Initial should be true on first request to receive one-time data.
	Initial bool `json:"initial,omitempty"`
}

// StatusResponse provides server status.
// For service discovery with type schemas, use Discovery.Describe instead.
type StatusResponse struct {
	// OK indicates the server is healthy.
	OK bool `json:"ok"`
	// Port is the server's listening port.
	Port int `json:"port"`
	// RawrData contains encoded data blobs (sent on initial request).
	RawrData []string `json:"rawrData,omitempty"`
}

// Status returns server status.
// For service discovery, use Discovery.Describe instead.
func (s *Service) Status(ctx context.Context, req *StatusRequest) (*StatusResponse, error) {
	resp := &StatusResponse{
		OK:   true,
		Port: s.port,
	}
	if req.Initial {
		resp.RawrData = loadRawrData()
	}
	return resp, nil
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
