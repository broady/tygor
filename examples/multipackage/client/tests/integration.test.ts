import { describe, test, expect } from "bun:test";
import type { v1_User, v2_User, MigrationRequest, MigrationResponse } from "../src/rpc/types";

describe("multipackage types", () => {
  test("v1_User has correct fields", () => {
    const user: v1_User = { id: 1, name: "Test" };
    expect(user.id).toBe(1);
    expect(user.name).toBe("Test");
  });

  test("v2_User has additional fields", () => {
    const user: v2_User = {
      id: 1,
      name: "Test",
      email: "test@example.com",
      created_at: "2024-01-01T00:00:00Z",
    };
    expect(user.id).toBe(1);
    expect(user.email).toBe("test@example.com");
    expect(user.created_at).toBe("2024-01-01T00:00:00Z");
  });

  test("MigrationRequest contains both user types", () => {
    const req: MigrationRequest = {
      v1_user: { id: 1, name: "Old" },
      v2_user: { id: 2, name: "New", email: "new@test.com", created_at: "2024-01-01" },
    };
    expect(req.v1_user.id).toBe(1);
    expect(req.v2_user.email).toBe("new@test.com");
  });

  test("MigrationResponse contains both user types", () => {
    const res: MigrationResponse = {
      success: true,
      v1_user: { id: 1, name: "Old" },
      v2_user: { id: 2, name: "New", email: "new@test.com", created_at: "2024-01-01" },
    };
    expect(res.success).toBe(true);
    expect(res.v1_user.name).toBe("Old");
    expect(res.v2_user.name).toBe("New");
  });

  test("types are properly disambiguated", () => {
    // This test verifies that v1_User and v2_User are different types
    // If they were both just "User", TypeScript would merge them incorrectly
    const v1: v1_User = { id: 1, name: "V1" };
    const v2: v2_User = { id: 2, name: "V2", email: "v2@test.com", created_at: "2024" };

    // v1 should NOT have email field (compile error if wrong)
    expect("email" in v1).toBe(false);

    // v2 MUST have email field
    expect("email" in v2).toBe(true);
    expect(v2.email).toBe("v2@test.com");
  });
});
