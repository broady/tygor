# Starter Solid example Makefile

GO_FILES := main.go types.go
TS_FILES := $(wildcard src/*.tsx src/*.ts src/rpc/*.ts)
GEN_DIR := ./src/rpc

include ../common.mk

# Run Solid dev server with Vite
dev: gen
	@bun install --silent
	@echo "Open http://localhost:5173 in your browser"
	@bun run dev
