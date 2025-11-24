# @tygor/client

TypeScript runtime client for the [tygor](https://github.com/broady/tygor) RPC framework.

## Installation

```bash
npm install @tygor/client
```

## Usage

This package provides the runtime client used with tygor-generated TypeScript types and service registry manifest.

### Example

```typescript
import { createClient } from '@tygor/client';
import { registry } from './generated/manifest';

const client = createClient(
  registry,
  {
    baseUrl: 'http://localhost:8080',
    headers: () => ({
      'Authorization': 'Bearer my-token'
    })
  }
);

// Type-safe RPC calls
const result = await client.MyService.MyMethod({ param: 'value' });
```

## Features

- **Lightweight**: Uses JavaScript Proxies for minimal bundle size
- **Type-safe**: Full TypeScript support with generated types
- **Simple**: Clean, idiomatic API with no code generation bloat
- **Modern**: Built on native `fetch` API

## Error Handling

The client throws `RPCError` instances for failed requests:

```typescript
import { RPCError } from '@tygor/client';

try {
  await client.MyService.MyMethod({ param: 'value' });
} catch (err) {
  if (err instanceof RPCError) {
    console.error(err.code);     // e.g., "invalid_argument"
    console.error(err.message);  // Human-readable message
    console.error(err.details);  // Additional error details
  }
}
```

## Documentation

For full documentation, see the [tygor repository](https://github.com/broady/tygor).

## License

MIT
