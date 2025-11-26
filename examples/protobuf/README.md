# Protobuf Example

This example demonstrates using protobuf-generated Go types as request/response types in tygor handlers.

## Overview

Tygor works seamlessly with protobuf messages - you can use proto-generated structs directly in your handlers. This is useful when you have existing protobuf definitions or want to share types across services.

## Regenerating Proto Types

If you modify `api/messages.proto`, regenerate the Go code:

```bash
protoc --go_out=. --go_opt=paths=source_relative api/messages.proto
```

## Running the Server

```bash
go run .
```

## Generating TypeScript Types

```bash
go run . -gen -out ./client/src/rpc
```

## API Endpoints

- `POST /Items/Create` - Create a new item
- `GET /Items/List` - List items with pagination
- `GET /Items/Get` - Get an item by ID
