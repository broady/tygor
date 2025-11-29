import { describe, test, expect, beforeAll, afterAll } from "bun:test";
import { createClient, ServerError, ValidationError } from "@tygor/client";
import { startServer, type RunningServer } from "@tygor/testing";
import { registry } from "../src/rpc/manifest";
import { schemaMap } from "../src/rpc/schemas.map.zod";

// [snippet:client-validation]

/**
 * This test demonstrates automatic client-side validation:
 * 1. Client validates request before sending (using schemaMap)
 * 2. Server validates with go-playground/validator
 *
 * Both use the same validation rules from Go struct tags.
 */

let server: RunningServer;

// Client WITH validation (request validation enabled by default when schemas provided)
let validatedClient: ReturnType<typeof createClient<typeof registry.manifest>>;

// Client WITHOUT validation (for testing server-side validation)
let rawClient: ReturnType<typeof createClient<typeof registry.manifest>>;

beforeAll(async () => {
  server = await startServer({
    cwd: new URL("../../", import.meta.url).pathname,
  });

  // Create client with automatic request validation
  validatedClient = createClient(registry, {
    baseUrl: server.url,
    schemas: schemaMap,
  });

  // Create client without validation (to test server behavior)
  rawClient = createClient(registry, { baseUrl: server.url });
});

afterAll(async () => {
  await server?.stop();
});

describe("Automatic client validation", () => {
  test("client throws ValidationError for invalid request", async () => {
    try {
      await validatedClient.Users.Create({
        username: "ab", // too short (min=3)
        email: "test@example.com",
        password: "password123",
      });
      expect.unreachable("Should have thrown ValidationError");
    } catch (e) {
      expect(e).toBeInstanceOf(ValidationError);
      if (e instanceof ValidationError) {
        expect(e.direction).toBe("request");
        expect(e.endpoint).toBe("Users.Create");
      }
    }
  });

  test("server-side validation still works for unvalidated clients", async () => {
    // rawClient doesn't validate, so request goes to server
    try {
      await rawClient.Users.Create({
        username: "ab", // too short - server will reject
        email: "test@example.com",
        password: "password123",
      });
      expect.unreachable("Should have thrown ServerError");
    } catch (e) {
      expect(e).toBeInstanceOf(ServerError);
      if (e instanceof ServerError) {
        expect(e.code).toBe("invalid_argument");
      }
    }
  });

  test("valid data passes both client and server validation", async () => {
    const user = await validatedClient.Users.Create({
      username: "validuser",
      email: "valid@example.com",
      password: "securepass123",
    });

    expect(user.username).toBe("validuser");
    expect(user.email).toBe("valid@example.com");
  });

  test("email validation catches invalid format", async () => {
    try {
      await validatedClient.Users.Create({
        username: "testuser",
        email: "not-an-email",
        password: "password123",
      });
      expect.unreachable("Should have thrown ValidationError");
    } catch (e) {
      expect(e).toBeInstanceOf(ValidationError);
    }
  });

  test("enum validation catches invalid values", async () => {
    try {
      await validatedClient.Tasks.Create({
        title: "Test task",
        priority: "urgent" as any, // not in enum
        tags: [],
      });
      expect.unreachable("Should have thrown ValidationError");
    } catch (e) {
      expect(e).toBeInstanceOf(ValidationError);
    }
  });
});

// [/snippet:client-validation]
