import { describe, test, expect, mock } from "bun:test";
import { createClient, TygorError, RPCError, TransportError, ServiceRegistry } from "./runtime";

// Helper to create a mock response
function mockResponse(status: number, body: any, statusText = "") {
  const bodyStr = typeof body === "string" ? body : JSON.stringify(body);
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText,
    clone: function() { return this; },
    text: async () => bodyStr,
  };
}

describe("TygorError hierarchy", () => {
  test("RPCError has correct properties", () => {
    const error = new RPCError("not_found", "Resource not found", 404);
    expect(error.name).toBe("RPCError");
    expect(error.kind).toBe("rpc");
    expect(error.code).toBe("not_found");
    expect(error.message).toBe("Resource not found");
    expect(error.httpStatus).toBe(404);
    expect(error.details).toBeUndefined();
    expect(error).toBeInstanceOf(TygorError);
    expect(error).toBeInstanceOf(RPCError);
  });

  test("RPCError with details", () => {
    const error = new RPCError("validation_error", "Invalid input", 400, {
      field: "email",
      reason: "invalid format",
    });
    expect(error.code).toBe("validation_error");
    expect(error.httpStatus).toBe(400);
    expect(error.details).toEqual({ field: "email", reason: "invalid format" });
  });

  test("TransportError has correct properties", () => {
    const error = new TransportError("Bad Gateway", 502, "<html>...</html>");
    expect(error.name).toBe("TransportError");
    expect(error.kind).toBe("transport");
    expect(error.message).toBe("Bad Gateway");
    expect(error.httpStatus).toBe(502);
    expect(error.rawBody).toBe("<html>...</html>");
    expect(error).toBeInstanceOf(TygorError);
    expect(error).toBeInstanceOf(TransportError);
  });

  test("can discriminate error types", () => {
    const rpcError: TygorError = new RPCError("not_found", "Not found", 404);
    const transportError: TygorError = new TransportError("Bad Gateway", 502);

    // instanceof narrowing
    if (rpcError instanceof RPCError) {
      expect(rpcError.code).toBe("not_found");
    }
    if (transportError instanceof TransportError) {
      expect(transportError.httpStatus).toBe(502);
    }

    // kind discriminant
    expect(rpcError.kind).toBe("rpc");
    expect(transportError.kind).toBe("transport");
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
    const mockFetch = mock(async () => mockResponse(200, { result: { data: "success" } }));

    const client = createClient(mockRegistry, {
      baseUrl: "http://localhost:8080",
      fetch: mockFetch,
    });

    const result = await client.Test.Get({ id: "123" });

    expect(result).toEqual({ data: "success" });
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/get?id=123",
      expect.objectContaining({ method: "GET" })
    );
  });

  test("successful POST request", async () => {
    const mockFetch = mock(async () => mockResponse(200, { result: { created: true } }));

    const client = createClient(mockRegistry, {
      baseUrl: "http://localhost:8080",
      fetch: mockFetch,
    });

    const result = await client.Test.Post({ name: "test" });

    expect(result).toEqual({ created: true });
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/post",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "Content-Type": "application/json" }),
        body: JSON.stringify({ name: "test" }),
      })
    );
  });

  test("GET request with custom headers", async () => {
    const mockFetch = mock(async () => mockResponse(200, { result: { data: "success" } }));

    const client = createClient(mockRegistry, {
      baseUrl: "http://localhost:8080",
      headers: () => ({ Authorization: "Bearer token123" }),
      fetch: mockFetch,
    });

    await client.Test.Get({ id: "123" });

    expect(mockFetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        headers: expect.objectContaining({ Authorization: "Bearer token123" }),
      })
    );
  });

  test("uses globalThis.fetch when fetch option is not provided", async () => {
    const mockFetch = mock(async () => mockResponse(200, { result: { data: "success" } }));
    global.fetch = mockFetch as any;

    const client = createClient(mockRegistry, { baseUrl: "http://localhost:8080" });

    const result = await client.Test.Get({ id: "123" });

    expect(result).toEqual({ data: "success" });
  });

  test("error response with valid JSON error envelope (RPCError)", async () => {
    const mockFetch = mock(async () => mockResponse(400, {
      error: {
        code: "invalid_input",
        message: "Email is required",
        details: { field: "email" },
      },
    }, "Bad Request"));
    global.fetch = mockFetch as any;

    const client = createClient(mockRegistry, { baseUrl: "http://localhost:8080" });

    try {
      await client.Test.Get({ id: "123" });
      expect.unreachable("Should have thrown an error");
    } catch (e: any) {
      expect(e).toBeInstanceOf(RPCError);
      expect(e.kind).toBe("rpc");
      expect(e.code).toBe("invalid_input");
      expect(e.message).toBe("Email is required");
      expect(e.httpStatus).toBe(400);
      expect(e.details).toEqual({ field: "email" });
    }
  });

  test("transport error with null body", async () => {
    const mockFetch = mock(async () => mockResponse(404, null, "Not Found"));
    global.fetch = mockFetch as any;

    const client = createClient(mockRegistry, { baseUrl: "http://localhost:8080" });

    try {
      await client.Test.Get({ id: "123" });
      expect.unreachable("Should have thrown an error");
    } catch (e: any) {
      expect(e).toBeInstanceOf(TransportError);
      expect(e.kind).toBe("transport");
      expect(e.httpStatus).toBe(404);
      expect(e.message).toBe("Invalid response format");
    }
  });

  test("transport error with invalid JSON (HTML from proxy)", async () => {
    const htmlBody = "<!DOCTYPE html><html><body>502 Bad Gateway</body></html>";
    const mockFetch = mock(async () => ({
      ok: false,
      status: 502,
      statusText: "Bad Gateway",
      clone: function() { return this; },
      text: async () => htmlBody,
    }));
    global.fetch = mockFetch as any;

    const client = createClient(mockRegistry, { baseUrl: "http://localhost:8080" });

    try {
      await client.Test.Get({ id: "123" });
      expect.unreachable("Should have thrown an error");
    } catch (e: any) {
      expect(e).toBeInstanceOf(TransportError);
      expect(e.kind).toBe("transport");
      expect(e.httpStatus).toBe(502);
      expect(e.message).toBe("Bad Gateway");
      expect(e.rawBody).toBe(htmlBody);
    }
  });

  test("error response with partial error object (missing code)", async () => {
    const mockFetch = mock(async () => mockResponse(400, {
      error: { message: "Something went wrong" },
    }, "Bad Request"));
    global.fetch = mockFetch as any;

    const client = createClient(mockRegistry, { baseUrl: "http://localhost:8080" });

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
    const mockFetch = mock(async () => mockResponse(403, {
      error: { code: "forbidden" },
    }, "Forbidden"));
    global.fetch = mockFetch as any;

    const client = createClient(mockRegistry, { baseUrl: "http://localhost:8080" });

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
    const client = createClient(mockRegistry, { baseUrl: "http://localhost:8080" });

    expect(() => {
      (client as any).Unknown.Method({ test: true });
    }).toThrow("Unknown RPC method: Unknown.Method");
  });

  test("GET request handles array parameters", async () => {
    const mockFetch = mock(async () => mockResponse(200, { result: { data: "success" } }));
    global.fetch = mockFetch as any;

    const metadata = { "Test.Search": { method: "GET", path: "/test/search" } };
    type SearchManifest = { "Test.Search": { req: { tags: string[] }; res: { data: string } } };
    const searchRegistry: ServiceRegistry<SearchManifest> = {
      manifest: {} as SearchManifest,
      metadata,
    };

    const client = createClient(searchRegistry, { baseUrl: "http://localhost:8080" });
    await client.Test.Search({ tags: ["foo", "bar"] });

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/search?tags=foo&tags=bar",
      expect.any(Object)
    );
  });

  test("GET request omits null and undefined parameters", async () => {
    const mockFetch = mock(async () => mockResponse(200, { result: { data: "success" } }));
    global.fetch = mockFetch as any;

    const metadata = { "Test.Query": { method: "GET", path: "/test/query" } };
    type QueryManifest = {
      "Test.Query": { req: { id: string; optional?: string; nullable: string | null }; res: { data: string } };
    };
    const queryRegistry: ServiceRegistry<QueryManifest> = {
      manifest: {} as QueryManifest,
      metadata,
    };

    const client = createClient(queryRegistry, { baseUrl: "http://localhost:8080" });
    await client.Test.Query({ id: "123", optional: undefined, nullable: null });

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/query?id=123",
      expect.any(Object)
    );
  });

  test("POST request with custom headers preserves Authorization header", async () => {
    const mockFetch = mock(async () => mockResponse(200, { result: { created: true } }));
    global.fetch = mockFetch as any;

    const client = createClient(mockRegistry, {
      baseUrl: "http://localhost:8080",
      headers: () => ({ Authorization: "Bearer token123" }),
    });

    await client.Test.Post({ name: "test" });

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/post",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({
          Authorization: "Bearer token123",
          "Content-Type": "application/json",
        }),
      })
    );
  });

  test("successful PUT request", async () => {
    const mockFetch = mock(async () => mockResponse(200, { result: { updated: true } }));
    global.fetch = mockFetch as any;

    const client = createClient(mockRegistry, { baseUrl: "http://localhost:8080" });
    const result = await client.Test.Put({ id: "123", name: "updated" });

    expect(result).toEqual({ updated: true });
  });

  test("successful PATCH request", async () => {
    const mockFetch = mock(async () => mockResponse(200, { result: { updated: true } }));
    global.fetch = mockFetch as any;

    const client = createClient(mockRegistry, { baseUrl: "http://localhost:8080" });
    const result = await client.Test.Patch({ id: "123", name: "patched" });

    expect(result).toEqual({ updated: true });
  });

  test("successful DELETE request", async () => {
    const mockFetch = mock(async () => mockResponse(200, { result: { deleted: true } }));
    global.fetch = mockFetch as any;

    const client = createClient(mockRegistry, { baseUrl: "http://localhost:8080" });
    const result = await client.Test.Delete({ id: "123" });

    expect(result).toEqual({ deleted: true });
  });

  test("HEAD request should use query parameters like GET", async () => {
    const mockFetch = mock(async () => mockResponse(200, { result: { exists: true } }));
    global.fetch = mockFetch as any;

    const metadata = { "Test.Head": { method: "HEAD", path: "/test/head" } };
    type HeadManifest = { "Test.Head": { req: { id: string }; res: { exists: boolean } } };
    const headRegistry: ServiceRegistry<HeadManifest> = {
      manifest: {} as HeadManifest,
      metadata,
    };

    const client = createClient(headRegistry, { baseUrl: "http://localhost:8080" });
    const result = await client.Test.Head({ id: "123" });

    expect(result).toEqual({ exists: true });
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/test/head?id=123",
      expect.objectContaining({ method: "HEAD" })
    );
  });

  test("GET request query parameters are sorted for consistent caching", async () => {
    const mockFetch = mock(async () => mockResponse(200, { result: { data: "success" } }));
    global.fetch = mockFetch as any;

    const metadata = { "Test.Search": { method: "GET", path: "/test/search" } };
    type SearchManifest = {
      "Test.Search": { req: { name: string; id: string; limit: number }; res: { data: string } };
    };
    const searchRegistry: ServiceRegistry<SearchManifest> = {
      manifest: {} as SearchManifest,
      metadata,
    };

    const client = createClient(searchRegistry, { baseUrl: "http://localhost:8080" });

    await client.Test.Search({ name: "test", id: "123", limit: 10 });
    const url1 = mockFetch.mock.calls[0][0];
    mockFetch.mockClear();

    await client.Test.Search({ limit: 10, id: "123", name: "test" });
    const url2 = mockFetch.mock.calls[0][0];

    expect(url1).toBe(url2);
    expect(url1).toBe("http://localhost:8080/test/search?id=123&limit=10&name=test");
  });

  test("null result response returns null (Empty type)", async () => {
    const mockFetch = mock(async () => mockResponse(200, { result: null }));
    global.fetch = mockFetch as any;

    const client = createClient(mockRegistry, { baseUrl: "http://localhost:8080" });
    const result = await client.Test.Delete({ id: "123" });

    expect(result).toBeNull();
  });

  test("transport error for malformed envelope without result or error field", async () => {
    const mockFetch = mock(async () => mockResponse(200, { foo: "bar" }));
    global.fetch = mockFetch as any;

    const client = createClient(mockRegistry, { baseUrl: "http://localhost:8080" });

    try {
      await client.Test.Get({ id: "123" });
      expect.unreachable("Should have thrown an error");
    } catch (e: any) {
      expect(e).toBeInstanceOf(TransportError);
      expect(e.kind).toBe("transport");
      expect(e.httpStatus).toBe(200);
      expect(e.message).toBe("Invalid response format: missing result or error field");
    }
  });

  test("transport error includes truncated rawBody for debugging", async () => {
    const longBody = "x".repeat(2000);
    const mockFetch = mock(async () => ({
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
      clone: function() { return this; },
      text: async () => longBody,
    }));
    global.fetch = mockFetch as any;

    const client = createClient(mockRegistry, { baseUrl: "http://localhost:8080" });

    try {
      await client.Test.Get({ id: "123" });
      expect.unreachable("Should have thrown an error");
    } catch (e: any) {
      expect(e).toBeInstanceOf(TransportError);
      expect(e.rawBody?.length).toBe(1000); // Truncated to 1000 chars
    }
  });
});
