# Protobuf Example

Demonstrates using protobuf-generated Go types as request/response types in tygor handlers.

## Overview

Tygor works seamlessly with protobuf messages - you can use proto-generated structs directly in your handlers. This is useful when you have existing protobuf definitions or want to share types across services.

## Running the Example

```bash
make run      # Start server on :8080
make gen      # Generate TypeScript types
make proto    # Regenerate Go code from .proto
```

## Code Overview

### Proto Definitions

<!-- [snippet:proto-messages] -->
```protobuf title="messages.proto"
// CreateItemRequest is the request to create an item.
message CreateItemRequest {
  string name = 1;
  string description = 2;
  int64 price_cents = 3;
  repeated string tags = 4;
}

// Item represents an item in the catalog.
message Item {
  int64 id = 1;
  string name = 2;
  string description = 3;
  int64 price_cents = 4;
  repeated string tags = 5;
}
```
<!-- [/snippet:proto-messages] -->

### Handler Using Proto Types

<!-- [snippet:proto-handler] -->
```go title="main.go"
func CreateItem(ctx context.Context, req *api.CreateItemRequest) (*api.Item, error) {
	if req.Name == "" {
		return nil, tygor.NewError(tygor.CodeInvalidArgument, "name is required")
	}

	id := idCounter.Add(1)
	return &api.Item{
		Id:          id,
		Name:        req.Name,
		Description: req.Description,
		PriceCents:  req.PriceCents,
		Tags:        req.Tags,
	}, nil
}

```
<!-- [/snippet:proto-handler] -->

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/Items/Create` | Create a new item |
| GET | `/Items/List` | List items with pagination |
| GET | `/Items/Get` | Get an item by ID |

## File Structure

```
protobuf/
├── main.go              # Server and handlers
├── api/
│   ├── messages.proto   # Protobuf definitions
│   └── messages.pb.go   # Generated Go code
├── client/src/rpc/      # Generated TypeScript
└── README.md
```
