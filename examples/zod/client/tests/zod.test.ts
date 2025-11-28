import { describe, test, expect } from "bun:test";
import {
  CreateUserRequestSchema,
  CreateTaskRequestSchema,
} from "../src/rpc/schemas.zod";

// [snippet:schema-generation]

/**
 * These tests verify that Zod schemas are correctly generated from Go
 * validation tags. The schemas can be used directly for manual validation,
 * or automatically via the client's schemaMap (see integration.test.ts).
 */

describe("Generated Zod schemas", () => {
  test("validates required fields", () => {
    const valid = CreateUserRequestSchema.safeParse({
      username: "alice123",
      email: "alice@example.com",
      password: "securepass",
    });
    expect(valid.success).toBe(true);

    const missing = CreateUserRequestSchema.safeParse({
      username: "alice123",
      // missing email and password
    });
    expect(missing.success).toBe(false);
  });

  test("validates string constraints (min/max)", () => {
    const tooShort = CreateUserRequestSchema.safeParse({
      username: "ab", // min=3
      email: "a@b.com",
      password: "12345678",
    });
    expect(tooShort.success).toBe(false);

    const tooLong = CreateUserRequestSchema.safeParse({
      username: "a".repeat(21), // max=20
      email: "a@b.com",
      password: "12345678",
    });
    expect(tooLong.success).toBe(false);
  });

  test("validates email format", () => {
    const invalid = CreateUserRequestSchema.safeParse({
      username: "alice123",
      email: "not-an-email",
      password: "securepass",
    });
    expect(invalid.success).toBe(false);
  });

  test("validates oneof/enum values", () => {
    const valid = CreateTaskRequestSchema.safeParse({
      title: "Test",
      priority: "high", // valid enum value
      tags: [],
    });
    expect(valid.success).toBe(true);

    const invalid = CreateTaskRequestSchema.safeParse({
      title: "Test",
      priority: "urgent", // not in oneof
      tags: [],
    });
    expect(invalid.success).toBe(false);
  });

  test("validates optional nullable fields", () => {
    // Optional fields can be omitted
    const withoutOptional = CreateUserRequestSchema.safeParse({
      username: "alice123",
      email: "alice@example.com",
      password: "securepass",
    });
    expect(withoutOptional.success).toBe(true);

    // Optional fields can be null
    const withNull = CreateUserRequestSchema.safeParse({
      username: "alice123",
      email: "alice@example.com",
      password: "securepass",
      website: null,
      age: null,
    });
    expect(withNull.success).toBe(true);

    // Optional fields are validated when provided
    const invalidUrl = CreateUserRequestSchema.safeParse({
      username: "alice123",
      email: "alice@example.com",
      password: "securepass",
      website: "not-a-url",
    });
    expect(invalidUrl.success).toBe(false);
  });
});

// [/snippet:schema-generation]
