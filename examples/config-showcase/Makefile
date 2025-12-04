# Config Showcase Example Makefile
# Custom generation logic - generates multiple output variants

GO_FILES := main.go api/types.go
TS_FILES := $(wildcard client/src/*/types.ts)
GEN_DIR := ./client/src

# Use PREGEN to run custom generation instead of default tygor gen
PREGEN := go run . gen
# Skip the default tygor gen step (: is shell no-op, \# comments out the args)
TYGOR := : \#

include ../common.mk
