import { describe, test, expect } from "bun:test";
import {
  CreateUserRequestSchema,
  CreateTaskRequestSchema,
  UpdateTaskRequestSchema,
  TaskSchema,
  UserSchema,
  ListParamsSchema,
  GetTaskParamsSchema,
} from "../src/rpc/schemas.zod";

// [snippet:basic-validation]

describe("CreateUserRequest validation", () => {
  test("accepts valid user data", () => {
    const result = CreateUserRequestSchema.safeParse({
      username: "alice123",
      email: "alice@example.com",
      password: "securepass",
    });

    expect(result.success).toBe(true);
  });

  test("rejects username too short", () => {
    const result = CreateUserRequestSchema.safeParse({
      username: "ab", // min 3
      email: "alice@example.com",
      password: "securepass",
    });

    expect(result.success).toBe(false);
  });

  test("rejects username too long", () => {
    const result = CreateUserRequestSchema.safeParse({
      username: "a".repeat(21), // max 20
      email: "alice@example.com",
      password: "securepass",
    });

    expect(result.success).toBe(false);
  });

  test("rejects non-alphanumeric username", () => {
    const result = CreateUserRequestSchema.safeParse({
      username: "alice_123", // alphanum only
      email: "alice@example.com",
      password: "securepass",
    });

    expect(result.success).toBe(false);
  });

  test("rejects invalid email", () => {
    const result = CreateUserRequestSchema.safeParse({
      username: "alice123",
      email: "not-an-email",
      password: "securepass",
    });

    expect(result.success).toBe(false);
  });

  test("rejects password too short", () => {
    const result = CreateUserRequestSchema.safeParse({
      username: "alice123",
      email: "alice@example.com",
      password: "short", // min 8
    });

    expect(result.success).toBe(false);
  });

  test("accepts optional fields", () => {
    const result = CreateUserRequestSchema.safeParse({
      username: "alice123",
      email: "alice@example.com",
      password: "securepass",
      website: "https://example.com",
      age: 25,
    });

    expect(result.success).toBe(true);
  });

  test("validates optional URL field", () => {
    const result = CreateUserRequestSchema.safeParse({
      username: "alice123",
      email: "alice@example.com",
      password: "securepass",
      website: "not-a-url",
    });

    expect(result.success).toBe(false);
  });

  test("validates age constraints", () => {
    const tooYoung = CreateUserRequestSchema.safeParse({
      username: "alice123",
      email: "alice@example.com",
      password: "securepass",
      age: 10, // min 13
    });
    expect(tooYoung.success).toBe(false);

    const tooOld = CreateUserRequestSchema.safeParse({
      username: "alice123",
      email: "alice@example.com",
      password: "securepass",
      age: 200, // max 150
    });
    expect(tooOld.success).toBe(false);
  });
});

// [/snippet:basic-validation]

// [snippet:oneof-validation]

describe("Priority oneof validation", () => {
  test("accepts valid priority values in task", () => {
    const priorities = ["low", "medium", "high", "critical"];
    for (const priority of priorities) {
      const result = CreateTaskRequestSchema.safeParse({
        title: "Test task",
        priority,
        tags: [],
      });
      expect(result.success).toBe(true);
    }
  });

  test("rejects invalid priority", () => {
    const result = CreateTaskRequestSchema.safeParse({
      title: "Test task",
      priority: "urgent", // not in oneof
      tags: [],
    });

    expect(result.success).toBe(false);
  });
});

// [/snippet:oneof-validation]

// [snippet:task-validation]

describe("CreateTaskRequest validation", () => {
  test("accepts valid task", () => {
    const result = CreateTaskRequestSchema.safeParse({
      title: "Write documentation",
      priority: "high",
      tags: ["docs", "important"],
    });

    expect(result.success).toBe(true);
  });

  test("rejects empty title", () => {
    const result = CreateTaskRequestSchema.safeParse({
      title: "", // min 1
      priority: "medium",
      tags: [],
    });

    expect(result.success).toBe(false);
  });

  test("rejects title too long", () => {
    const result = CreateTaskRequestSchema.safeParse({
      title: "x".repeat(201), // max 200
      priority: "medium",
      tags: [],
    });

    expect(result.success).toBe(false);
  });

  test("validates description max length", () => {
    const result = CreateTaskRequestSchema.safeParse({
      title: "Test task",
      description: "x".repeat(2001), // max 2000
      priority: "medium",
      tags: [],
    });

    expect(result.success).toBe(false);
  });

  test("validates assignee_id must be positive", () => {
    const result = CreateTaskRequestSchema.safeParse({
      title: "Test task",
      priority: "medium",
      assignee_id: 0, // gt 0
      tags: [],
    });

    expect(result.success).toBe(false);
  });
});

// [/snippet:task-validation]

// [snippet:update-validation]

describe("UpdateTaskRequest validation", () => {
  test("requires task_id", () => {
    const result = UpdateTaskRequestSchema.safeParse({
      title: "Updated title",
    });

    expect(result.success).toBe(false);
  });

  test("validates task_id is positive", () => {
    const result = UpdateTaskRequestSchema.safeParse({
      task_id: -1, // gt 0
    });

    expect(result.success).toBe(false);
  });

  test("accepts partial updates", () => {
    const result = UpdateTaskRequestSchema.safeParse({
      task_id: 1,
      completed: true,
    });

    expect(result.success).toBe(true);
  });

  test("validates optional title constraints", () => {
    const tooShort = UpdateTaskRequestSchema.safeParse({
      task_id: 1,
      title: "", // min 1
    });
    expect(tooShort.success).toBe(false);

    const tooLong = UpdateTaskRequestSchema.safeParse({
      task_id: 1,
      title: "x".repeat(201), // max 200
    });
    expect(tooLong.success).toBe(false);
  });

  test("validates optional priority is valid enum value", () => {
    const invalid = UpdateTaskRequestSchema.safeParse({
      task_id: 1,
      priority: "invalid",
    });
    expect(invalid.success).toBe(false);

    const valid = UpdateTaskRequestSchema.safeParse({
      task_id: 1,
      priority: "high",
    });
    expect(valid.success).toBe(true);
  });
});

// [/snippet:update-validation]

// [snippet:pagination-validation]

describe("ListParams pagination validation", () => {
  test("accepts valid pagination", () => {
    const result = ListParamsSchema.safeParse({
      limit: 10,
      offset: 0,
    });
    expect(result.success).toBe(true);
  });

  test("rejects limit too low", () => {
    const result = ListParamsSchema.safeParse({
      limit: 0, // gte 1
      offset: 0,
    });
    expect(result.success).toBe(false);
  });

  test("rejects limit too high", () => {
    const result = ListParamsSchema.safeParse({
      limit: 101, // lte 100
      offset: 0,
    });
    expect(result.success).toBe(false);
  });

  test("rejects negative offset", () => {
    const result = ListParamsSchema.safeParse({
      limit: 10,
      offset: -1, // gte 0
    });
    expect(result.success).toBe(false);
  });
});

describe("GetTaskParams validation", () => {
  test("requires positive task_id", () => {
    const zero = GetTaskParamsSchema.safeParse({ task_id: 0 });
    expect(zero.success).toBe(false);

    const negative = GetTaskParamsSchema.safeParse({ task_id: -1 });
    expect(negative.success).toBe(false);

    const valid = GetTaskParamsSchema.safeParse({ task_id: 1 });
    expect(valid.success).toBe(true);
  });
});

// [/snippet:pagination-validation]

// [snippet:response-validation]

describe("Response schema validation", () => {
  test("validates User response", () => {
    const result = UserSchema.safeParse({
      id: 1,
      username: "alice123",
      email: "alice@example.com",
      created_at: "2024-01-15T10:30:00Z",
    });

    expect(result.success).toBe(true);
  });

  test("validates Task response", () => {
    const result = TaskSchema.safeParse({
      id: 1,
      title: "Test task",
      priority: "high",
      tags: ["test"],
      completed: false,
    });

    expect(result.success).toBe(true);
  });

  test("validates Task with all fields", () => {
    const result = TaskSchema.safeParse({
      id: 1,
      title: "Test task",
      description: "A test description",
      priority: "high",
      assignee_id: 42,
      tags: ["test", "example"],
      due_date: "2024-12-31T23:59:59Z",
      completed: false,
    });

    expect(result.success).toBe(true);
  });
});

// [/snippet:response-validation]

// [snippet:type-inference]

describe("Type inference", () => {
  test("inferred types work correctly", () => {
    // This test verifies TypeScript type inference works
    const validUser = CreateUserRequestSchema.parse({
      username: "alice123",
      email: "alice@example.com",
      password: "securepass",
    });

    // TypeScript knows the shape of validUser
    const username: string = validUser.username;
    const email: string = validUser.email;
    const website: string | null | undefined = validUser.website;

    expect(username).toBe("alice123");
    expect(email).toBe("alice@example.com");
    expect(website).toBeUndefined();
  });

  test("validated task has correct types", () => {
    const task = CreateTaskRequestSchema.parse({
      title: "Test",
      priority: "high",
      tags: ["a", "b"],
    });

    // TypeScript infers these types
    const title: string = task.title;
    const priority: "low" | "medium" | "high" | "critical" = task.priority;
    const tags: string[] = task.tags;

    expect(title).toBe("Test");
    expect(priority).toBe("high");
    expect(tags).toEqual(["a", "b"]);
  });
});

// [/snippet:type-inference]
