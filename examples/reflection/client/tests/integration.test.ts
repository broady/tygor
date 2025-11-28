import { describe, test, expect, beforeAll, afterAll } from "bun:test";
import { createClient, ServerError } from "@tygor/client";
import { startServer, type RunningServer } from "@tygor/testing";
import { registry } from "../src/rpc/manifest";

let server: RunningServer;
let client: ReturnType<typeof createClient<typeof registry.manifest>>;

beforeAll(async () => {
  server = await startServer({
    cwd: new URL("../../", import.meta.url).pathname,
  });

  client = createClient(registry, {
    baseUrl: server.url,
  });
});

afterAll(async () => {
  await server?.stop();
});

describe("Reflection provider generics", () => {
  describe("PagedResponse[User]", () => {
    test("returns properly typed paginated user response", async () => {
      const response = await client.Users.List({
        page: 1,
        page_size: 10,
      });

      // Verify PagedResponse structure
      expect(response).toHaveProperty("data");
      expect(response).toHaveProperty("total");
      expect(response).toHaveProperty("page");
      expect(response).toHaveProperty("page_size");
      expect(response).toHaveProperty("has_more");

      // Verify pagination metadata
      expect(response.page).toBe(1);
      expect(response.page_size).toBe(10);
      expect(typeof response.total).toBe("number");
      expect(typeof response.has_more).toBe("boolean");

      // Verify User array
      expect(Array.isArray(response.data)).toBe(true);
      if (response.data && response.data.length > 0) {
        const user = response.data[0];
        expect(user).toHaveProperty("id");
        expect(user).toHaveProperty("username");
        expect(user).toHaveProperty("email");
        expect(user).toHaveProperty("role");
      }
    });

    test("supports role filtering", async () => {
      const response = await client.Users.List({
        page: 1,
        page_size: 10,
        role: "admin",
      });

      expect(Array.isArray(response.data)).toBe(true);
      // All returned users should be admins
      if (response.data) {
        response.data.forEach(user => {
          expect(user.role).toBe("admin");
        });
      }
    });
  });

  describe("Result[User]", () => {
    test("returns success result with user data", async () => {
      const result = await client.Users.Get({ id: 1 });

      expect(result).toHaveProperty("success");
      expect(result.success).toBe(true);
      expect(result.data).toBeDefined();

      if (result.data) {
        expect(result.data).toHaveProperty("id");
        expect(result.data).toHaveProperty("username");
        expect(result.data).toHaveProperty("email");
        expect(result.data).toHaveProperty("role");
      }
    });

    test("returns error result for non-existent user", async () => {
      const result = await client.Users.Get({ id: 999 });

      expect(result).toHaveProperty("success");
      expect(result.success).toBe(false);
      expect(result.error).toBeDefined();
      expect(result.data).toBeUndefined();
    });
  });

  describe("Result[Post]", () => {
    test("returns success result with post data", async () => {
      const result = await client.Posts.Create({
        title: "Test Post",
        content: "This is a test post demonstrating generic types",
        author_id: 1,
      });

      expect(result).toHaveProperty("success");
      expect(result.success).toBe(true);
      expect(result.data).toBeDefined();

      if (result.data) {
        expect(result.data).toHaveProperty("id");
        expect(result.data).toHaveProperty("title");
        expect(result.data).toHaveProperty("content");
        expect(result.data).toHaveProperty("author_id");
        expect(result.data.title).toBe("Test Post");
        expect(result.data.author_id).toBe(1);
      }
    });
  });
});

describe("Type safety verification", () => {
  test("generic types maintain proper structure", async () => {
    // PagedResponse[User] has correct shape
    const usersPage = await client.Users.List({
      page: 1,
      page_size: 5,
    });

    // TypeScript enforces the structure
    const pageNumber: number = usersPage.page;
    const hasMore: boolean = usersPage.has_more;
    const users = usersPage.data;

    expect(pageNumber).toBeDefined();
    expect(typeof hasMore).toBe("boolean");
    expect(Array.isArray(users)).toBe(true);
  });

  test("Result type handles both success and error cases", async () => {
    const successResult = await client.Users.Get({ id: 1 });
    const errorResult = await client.Users.Get({ id: 999 });

    // TypeScript allows checking success/error
    if (successResult.success) {
      // data is available when success is true
      expect(successResult.data).toBeDefined();
    }

    if (!errorResult.success) {
      // error is available when success is false
      expect(errorResult.error).toBeDefined();
    }
  });
});
