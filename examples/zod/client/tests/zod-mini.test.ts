import { describe, test, expect } from "bun:test";
import {
  CreateUserRequestSchema,
  CreateTaskRequestSchema,
} from "../src/rpc/schemas.zod-mini";

/**
 * These tests verify that Zod-mini schemas are correctly generated from Go
 * validation tags. Zod-mini uses a functional API for tree-shaking benefits.
 */

describe("Generated Zod-mini schemas", () => {
  test("validates required fields", () => {
    const valid = CreateUserRequestSchema.parse({
      username: "alice123",
      email: "alice@example.com",
      password: "securepass",
    });
    expect(valid.username).toBe("alice123");

    expect(() =>
      CreateUserRequestSchema.parse({
        username: "alice123",
        // missing email and password
      })
    ).toThrow();
  });

  test("validates string constraints (min/max)", () => {
    expect(() =>
      CreateUserRequestSchema.parse({
        username: "ab", // min=3
        email: "a@b.com",
        password: "12345678",
      })
    ).toThrow();

    expect(() =>
      CreateUserRequestSchema.parse({
        username: "a".repeat(21), // max=20
        email: "a@b.com",
        password: "12345678",
      })
    ).toThrow();
  });

  test("validates email format", () => {
    expect(() =>
      CreateUserRequestSchema.parse({
        username: "alice123",
        email: "not-an-email",
        password: "securepass",
      })
    ).toThrow();
  });

  test("validates oneof/enum values", () => {
    const valid = CreateTaskRequestSchema.parse({
      title: "Test",
      priority: "high", // valid enum value
      tags: [],
    });
    expect(valid.priority).toBe("high");

    expect(() =>
      CreateTaskRequestSchema.parse({
        title: "Test",
        priority: "urgent", // not in oneof
        tags: [],
      })
    ).toThrow();
  });

  test("validates optional nullable fields", () => {
    // Optional fields can be omitted
    const withoutOptional = CreateUserRequestSchema.parse({
      username: "alice123",
      email: "alice@example.com",
      password: "securepass",
    });
    expect(withoutOptional.website).toBeUndefined();

    // Optional fields can be null
    const withNull = CreateUserRequestSchema.parse({
      username: "alice123",
      email: "alice@example.com",
      password: "securepass",
      website: null,
      age: null,
    });
    expect(withNull.website).toBeNull();

    // Optional fields are validated when provided
    expect(() =>
      CreateUserRequestSchema.parse({
        username: "alice123",
        email: "alice@example.com",
        password: "securepass",
        website: "not-a-url",
      })
    ).toThrow();
  });
});
