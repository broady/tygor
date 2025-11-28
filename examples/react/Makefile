# React example Makefile

GO_FILES := main.go api/types.go
TS_FILES := $(wildcard client/src/*.tsx client/src/rpc/*.ts)

include ../common.mk


# Run Vite dev server (run 'make run' in another terminal for the Go server)
dev: gen
	@(cd client && bun install --silent)
	@echo "Starting Vite... (run 'make run' in another terminal for the Go server)"
	@(cd client && bun run dev)
