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
├── client/
│   ├── index.ts         # TypeScript client example
│   └── src/rpc/         # Generated TypeScript types
└── README.md
```

## TypeScript Client

### Client Setup

<!-- [snippet:client-setup] -->
```typescript title="index.ts"
const client = createClient(registry, {
  baseUrl: "http://localhost:8080",
});
```
<!-- [/snippet:client-setup] -->

### Making RPC Calls

<!-- [snippet:client-calls] -->
```typescript title="index.ts"
// Create an item
const item = await client.Items.Create({
  name: "Widget",
  description: "A useful widget",
  price_cents: 1999,
  tags: ["gadget", "tool"],
});
console.log("Created item:", item.id, item.name);

// List items with pagination
const list = await client.Items.List({
  limit: 10,
  offset: 0,
});
console.log(`Found ${list.total} items:`);
list.items?.forEach((item) => {
  if (item) {
    console.log(`  - ${item.name}: $${(item.price_cents ?? 0) / 100}`);
  }
});

// Get a specific item
const fetched = await client.Items.Get({ id: item.id });
console.log("Fetched:", fetched.name, fetched.tags);
```
<!-- [/snippet:client-calls] -->

### Error Handling

<!-- [snippet:client-errors] -->
```typescript title="index.ts"
// Error handling example
async function handleErrors() {
  try {
    await client.Items.Get({ id: 99999 });
  } catch (e) {
    if (e instanceof RPCError) {
      console.error(`Error [${e.code}]: ${e.message}`);
    } else {
      throw e;
    }
  }
}
```
<!-- [/snippet:client-errors] -->
