# Devtools example Makefile

GO_FILES := main.go api/types.go
TS_FILES := $(wildcard client/src/*.tsx client/src/rpc/*.ts)

include ../common.mk


# Run Go server (with hot-reload) and Vite dev server in parallel
# - Vite plugin watches Go files, runs tygorgen, rebuilds
# - Vite proxies API requests to Go and picks up TS changes via HMR
dev: gen
	@bun install --silent
	@(cd client && bun install --silent)
	@echo "Open http://localhost:5173 in your browser"
	@bun run dev
