import { describe, test, expect, beforeAll, afterAll } from "bun:test";
import { createClient, ServerError } from "@tygor/client";
import { startServer, type RunningServer } from "@tygor/testing";
import { registry } from "../src/rpc/manifest";
import { CreateUserRequestSchema, CreateTaskRequestSchema } from "../src/rpc/schemas.zod";

// [snippet:two-way-validation]

/**
 * This test demonstrates two-way validation:
 * 1. Client-side: Zod validates before sending request
 * 2. Server-side: go-playground/validator validates in Go
 *
 * Both use the same validation rules from Go struct tags.
 */

let server: RunningServer;
let client: ReturnType<typeof createClient<typeof registry.manifest>>;

beforeAll(async () => {
  server = await startServer({
    cwd: new URL("../../", import.meta.url).pathname,
  });
  client = createClient(registry, { baseUrl: server.url });
});

afterAll(async () => {
  await server?.stop();
});

describe("Two-way validation", () => {
  test("client-side Zod catches invalid username before request", () => {
    // Zod validation happens on the client - no network request made
    const result = CreateUserRequestSchema.safeParse({
      username: "ab", // too short (min=3)
      email: "test@example.com",
      password: "password123",
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0].path).toContain("username");
    }
  });

  test("server-side go-playground/validator rejects invalid data", async () => {
    // Even if client skips validation, server enforces the same rules
    try {
      await client.Users.Create({
        username: "ab", // too short (min=3) - TypeScript allows it, server rejects
        email: "test@example.com",
        password: "password123",
      });
      expect.unreachable("Should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(ServerError);
      if (e instanceof ServerError) {
        // Server returns invalid_argument for validation failures
        expect(e.code).toBe("invalid_argument");
      }
    }
  });

  test("both client and server accept valid data", async () => {
    const validData = {
      username: "validuser",
      email: "valid@example.com",
      password: "securepass123",
    };

    // Client-side Zod validation passes
    const clientResult = CreateUserRequestSchema.safeParse(validData);
    expect(clientResult.success).toBe(true);

    // Server-side validation also passes
    const user = await client.Users.Create(validData);
    expect(user.username).toBe("validuser");
    expect(user.email).toBe("valid@example.com");
  });

  test("email validation works on both sides", async () => {
    const invalidEmail = {
      username: "testuser",
      email: "not-an-email",
      password: "password123",
    };

    // Client-side Zod catches it
    const clientResult = CreateUserRequestSchema.safeParse(invalidEmail);
    expect(clientResult.success).toBe(false);

    // Server-side also rejects it
    try {
      await client.Users.Create(invalidEmail);
      expect.unreachable("Should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(ServerError);
      if (e instanceof ServerError) {
        expect(e.code).toBe("invalid_argument");
      }
    }
  });

  test("oneof/enum validation on both sides", async () => {
    // Client-side: Zod validates priority enum
    const invalidPriority = CreateTaskRequestSchema.safeParse({
      title: "Test task",
      priority: "urgent", // not in oneof
      tags: [],
    });
    expect(invalidPriority.success).toBe(false);

    // Valid priority works
    const validTask = CreateTaskRequestSchema.safeParse({
      title: "Test task",
      priority: "high",
      tags: [],
    });
    expect(validTask.success).toBe(true);
  });
});

// [/snippet:two-way-validation]
