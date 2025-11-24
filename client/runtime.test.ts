import { describe, test, expect, mock } from "bun:test";
import { createClient, RPCError, ServiceRegistry } from "./runtime";

describe("RPCError", () => {
  test("creates error with code and message", () => {
    const error = new RPCError("not_found", "Resource not found");
    expect(error.name).toBe("RPCError");
    expect(error.code).toBe("not_found");
    expect(error.message).toBe("Resource not found");
    expect(error.details).toBeUndefined();
  });

  test("creates error with details", () => {
    const error = new RPCError("validation_error", "Invalid input", {
      field: "email",
      reason: "invalid format",
    });
    expect(error.code).toBe("validation_error");
    expect(error.details).toEqual({ field: "email", reason: "invalid format" });
  });
});

describe("createClient", () => {
  const mockMetadata = {
    "Test.Get": { method: "GET", path: "/test/get" },
    "Test.Post": { method: "POST", path: "/test/post" },
    "Test.Put": { method: "PUT", path: "/test/put" },
    "Test.Patch": { method: "PATCH", path: "/test/patch" },
    "Test.Delete": { method: "DELETE", path: "/test/delete" },
  };

  type TestManifest = {
    "Test.Get": { req: { id: string }; res: { data: string } };
    "Test.Post": { req: { name: string }; res: { created: boolean } };
    "Test.Put": { req: { id: string; name: string }; res: { updated: boolean } };
    "Test.Patch": { req: { id: string; name?: string }; res: { updated: boolean } };
    "Test.Delete": { req: { id: string }; res: { deleted: boolean } };
  };

  const mockRegistry: ServiceRegistry<TestManifest> = {
    manifest: {} as TestManifest,
    metadata: mockMetadata,
  };

  test("successful GET request", async () => {
    const mockFetch = mock(async () => ({
      ok: true,
      json: async () => ({ data: "success" }),
    }));
    global.fetch = mockFetch as any;

    const client = createClient(
      mockRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    const result = await client.Test.Get({ id: "123" });

    expect(result).toEqual({ data: "success" });
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/get?id=123",
      expect.objectContaining({
        method: "GET",
      })
    );
  });

  test("successful POST request", async () => {
    const mockFetch = mock(async () => ({
      ok: true,
      json: async () => ({ created: true }),
    }));
    global.fetch = mockFetch as any;

    const client = createClient(
      mockRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    const result = await client.Test.Post({ name: "test" });

    expect(result).toEqual({ created: true });
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/post",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({
          "Content-Type": "application/json",
        }),
        body: JSON.stringify({ name: "test" }),
      })
    );
  });

  test("GET request with custom headers", async () => {
    const mockFetch = mock(async () => ({
      ok: true,
      json: async () => ({ data: "success" }),
    }));
    global.fetch = mockFetch as any;

    const client = createClient(
      mockRegistry,
      {
        baseUrl: "http://localhost:8080",
        headers: () => ({ Authorization: "Bearer token123" }),
      }
    );

    await client.Test.Get({ id: "123" });

    expect(mockFetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer token123",
        }),
      })
    );
  });

  test("error response with valid JSON error object", async () => {
    const mockFetch = mock(async () => ({
      ok: false,
      statusText: "Bad Request",
      json: async () => ({
        code: "invalid_input",
        message: "Email is required",
        details: { field: "email" },
      }),
    }));
    global.fetch = mockFetch as any;

    const client = createClient(
      mockRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    try {
      await client.Test.Get({ id: "123" });
      expect.unreachable("Should have thrown an error");
    } catch (e: any) {
      expect(e).toBeInstanceOf(RPCError);
      expect(e.code).toBe("invalid_input");
      expect(e.message).toBe("Email is required");
      expect(e.details).toEqual({ field: "email" });
    }
  });

  test("error response with null body (empty response)", async () => {
    const mockFetch = mock(async () => ({
      ok: false,
      statusText: "Not Found",
      json: async () => null,
    }));
    global.fetch = mockFetch as any;

    const client = createClient(
      mockRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    try {
      await client.Test.Get({ id: "123" });
      expect.unreachable("Should have thrown an error");
    } catch (e: any) {
      expect(e).toBeInstanceOf(RPCError);
      expect(e.code).toBe("unknown");
      expect(e.message).toBe("Not Found");
    }
  });

  test("error response with undefined body", async () => {
    const mockFetch = mock(async () => ({
      ok: false,
      statusText: "Internal Server Error",
      json: async () => undefined,
    }));
    global.fetch = mockFetch as any;

    const client = createClient(
      mockRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    try {
      await client.Test.Get({ id: "123" });
      expect.unreachable("Should have thrown an error");
    } catch (e: any) {
      expect(e).toBeInstanceOf(RPCError);
      expect(e.code).toBe("unknown");
      expect(e.message).toBe("Internal Server Error");
    }
  });

  test("error response with non-object body", async () => {
    const mockFetch = mock(async () => ({
      ok: false,
      statusText: "Bad Gateway",
      json: async () => "some string error",
    }));
    global.fetch = mockFetch as any;

    const client = createClient(
      mockRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    try {
      await client.Test.Get({ id: "123" });
      expect.unreachable("Should have thrown an error");
    } catch (e: any) {
      expect(e).toBeInstanceOf(RPCError);
      expect(e.code).toBe("unknown");
      expect(e.message).toBe("Bad Gateway");
    }
  });

  test("error response with invalid JSON", async () => {
    const mockFetch = mock(async () => ({
      ok: false,
      statusText: "Service Unavailable",
      json: async () => {
        throw new Error("Invalid JSON");
      },
    }));
    global.fetch = mockFetch as any;

    const client = createClient(
      mockRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    try {
      await client.Test.Get({ id: "123" });
      expect.unreachable("Should have thrown an error");
    } catch (e: any) {
      expect(e).toBeInstanceOf(RPCError);
      expect(e.code).toBe("internal");
      expect(e.message).toBe("Service Unavailable");
    }
  });

  test("error response with partial error object (missing code)", async () => {
    const mockFetch = mock(async () => ({
      ok: false,
      statusText: "Bad Request",
      json: async () => ({
        message: "Something went wrong",
      }),
    }));
    global.fetch = mockFetch as any;

    const client = createClient(
      mockRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    try {
      await client.Test.Get({ id: "123" });
      expect.unreachable("Should have thrown an error");
    } catch (e: any) {
      expect(e).toBeInstanceOf(RPCError);
      expect(e.code).toBe("unknown");
      expect(e.message).toBe("Something went wrong");
    }
  });

  test("error response with partial error object (missing message)", async () => {
    const mockFetch = mock(async () => ({
      ok: false,
      statusText: "Forbidden",
      json: async () => ({
        code: "forbidden",
      }),
    }));
    global.fetch = mockFetch as any;

    const client = createClient(
      mockRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    try {
      await client.Test.Get({ id: "123" });
      expect.unreachable("Should have thrown an error");
    } catch (e: any) {
      expect(e).toBeInstanceOf(RPCError);
      expect(e.code).toBe("forbidden");
      expect(e.message).toBe("Unknown error");
    }
  });

  test("throws error for unknown RPC method", () => {
    const client = createClient(
      mockRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    expect(() => {
      (client as any).Unknown.Method({ test: true });
    }).toThrow("Unknown RPC method: Unknown.Method");
  });

  test("GET request handles array parameters", async () => {
    const mockFetch = mock(async () => ({
      ok: true,
      json: async () => ({ data: "success" }),
    }));
    global.fetch = mockFetch as any;

    const metadata = {
      "Test.Search": { method: "GET", path: "/test/search" },
    };

    type SearchManifest = {
      "Test.Search": { req: { tags: string[] }; res: { data: string } };
    };

    const searchRegistry: ServiceRegistry<SearchManifest> = {
      manifest: {} as SearchManifest,
      metadata,
    };

    const client = createClient(
      searchRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    await client.Test.Search({ tags: ["foo", "bar"] });

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/search?tags=foo&tags=bar",
      expect.any(Object)
    );
  });

  test("GET request omits null and undefined parameters", async () => {
    const mockFetch = mock(async () => ({
      ok: true,
      json: async () => ({ data: "success" }),
    }));
    global.fetch = mockFetch as any;

    const metadata = {
      "Test.Query": { method: "GET", path: "/test/query" },
    };

    type QueryManifest = {
      "Test.Query": {
        req: { id: string; optional?: string; nullable: string | null };
        res: { data: string };
      };
    };

    const queryRegistry: ServiceRegistry<QueryManifest> = {
      manifest: {} as QueryManifest,
      metadata,
    };

    const client = createClient(
      queryRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    await client.Test.Query({ id: "123", optional: undefined, nullable: null });

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/query?id=123",
      expect.any(Object)
    );
  });

  test("POST request with custom headers preserves Authorization header", async () => {
    const mockFetch = mock(async () => ({
      ok: true,
      json: async () => ({ created: true }),
    }));
    global.fetch = mockFetch as any;

    const client = createClient(
      mockRegistry,
      {
        baseUrl: "http://localhost:8080",
        headers: () => ({ Authorization: "Bearer token123" }),
      }
    );

    await client.Test.Post({ name: "test" });

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/post",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({
          Authorization: "Bearer token123",
          "Content-Type": "application/json",
        }),
        body: JSON.stringify({ name: "test" }),
      })
    );
  });

  test("successful PUT request", async () => {
    const mockFetch = mock(async () => ({
      ok: true,
      json: async () => ({ updated: true }),
    }));
    global.fetch = mockFetch as any;

    const client = createClient(
      mockRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    const result = await client.Test.Put({ id: "123", name: "updated" });

    expect(result).toEqual({ updated: true });
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/put",
      expect.objectContaining({
        method: "PUT",
        headers: expect.objectContaining({
          "Content-Type": "application/json",
        }),
        body: JSON.stringify({ id: "123", name: "updated" }),
      })
    );
  });

  test("successful PATCH request", async () => {
    const mockFetch = mock(async () => ({
      ok: true,
      json: async () => ({ updated: true }),
    }));
    global.fetch = mockFetch as any;

    const client = createClient(
      mockRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    const result = await client.Test.Patch({ id: "123", name: "patched" });

    expect(result).toEqual({ updated: true });
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/patch",
      expect.objectContaining({
        method: "PATCH",
        headers: expect.objectContaining({
          "Content-Type": "application/json",
        }),
        body: JSON.stringify({ id: "123", name: "patched" }),
      })
    );
  });

  test("successful DELETE request", async () => {
    const mockFetch = mock(async () => ({
      ok: true,
      json: async () => ({ deleted: true }),
    }));
    global.fetch = mockFetch as any;

    const client = createClient(
      mockRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    const result = await client.Test.Delete({ id: "123" });

    expect(result).toEqual({ deleted: true });
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/delete",
      expect.objectContaining({
        method: "DELETE",
        headers: expect.objectContaining({
          "Content-Type": "application/json",
        }),
        body: JSON.stringify({ id: "123" }),
      })
    );
  });

  test("HEAD request should use query parameters like GET", async () => {
    const mockFetch = mock(async () => ({
      ok: true,
      json: async () => ({ exists: true }),
    }));
    global.fetch = mockFetch as any;

    const metadata = {
      "Test.Head": { method: "HEAD", path: "/test/head" },
    };

    type HeadManifest = {
      "Test.Head": { req: { id: string }; res: { exists: boolean } };
    };

    const headRegistry: ServiceRegistry<HeadManifest> = {
      manifest: {} as HeadManifest,
      metadata,
    };

    const client = createClient(
      headRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    const result = await client.Test.Head({ id: "123" });

    expect(result).toEqual({ exists: true });
    // HEAD should use query params, not body
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/head?id=123",
      expect.objectContaining({
        method: "HEAD",
      })
    );
    // HEAD should NOT have Content-Type or body
    const call = mockFetch.mock.calls[0];
    expect(call[1].headers["Content-Type"]).toBeUndefined();
    expect(call[1].body).toBeUndefined();
  });

  test("GET request query parameters are sorted for consistent caching", async () => {
    const mockFetch = mock(async () => ({
      ok: true,
      json: async () => ({ data: "success" }),
    }));
    global.fetch = mockFetch as any;

    const metadata = {
      "Test.Search": { method: "GET", path: "/test/search" },
    };

    type SearchManifest = {
      "Test.Search": {
        req: { name: string; id: string; limit: number };
        res: { data: string };
      };
    };

    const searchRegistry: ServiceRegistry<SearchManifest> = {
      manifest: {} as SearchManifest,
      metadata,
    };

    const client = createClient(
      searchRegistry,
      { baseUrl: "http://localhost:8080" }
    );

    // Call with properties in different orders
    await client.Test.Search({ name: "test", id: "123", limit: 10 });
    const url1 = mockFetch.mock.calls[0][0];
    mockFetch.mockClear();

    await client.Test.Search({ limit: 10, id: "123", name: "test" });
    const url2 = mockFetch.mock.calls[0][0];
    mockFetch.mockClear();

    await client.Test.Search({ id: "123", limit: 10, name: "test" });
    const url3 = mockFetch.mock.calls[0][0];

    // All three calls should produce the same URL for consistent caching
    expect(url1).toBe(url2);
    expect(url2).toBe(url3);
    // Verify the URL has sorted parameters
    expect(url1).toBe("http://localhost:8080/test/search?id=123&limit=10&name=test");
  });
});
