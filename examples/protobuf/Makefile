# Protobuf example Makefile

GO_FILES := main.go api/messages.proto
TS_FILES := $(wildcard client/src/rpc/*.ts)

include ../common.mk

# Additional target for regenerating protobuf
.PHONY: proto

proto:
	protoc --go_out=. --go_opt=paths=source_relative api/messages.proto
