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

describe("Items service (protobuf types)", () => {
  describe("Items.List", () => {
    test("returns paginated list of items", async () => {
      const result = await client.Items.List({ limit: 10, offset: 0 });

      expect(result.items).toBeDefined();
      expect(Array.isArray(result.items)).toBe(true);
      expect(result.total).toBeGreaterThan(0);
    });

    test("pagination works correctly", async () => {
      const page1 = await client.Items.List({ limit: 1, offset: 0 });
      const page2 = await client.Items.List({ limit: 1, offset: 1 });

      expect(page1.items?.length).toBe(1);
      expect(page2.items?.length).toBe(1);

      if (page1.items?.[0] && page2.items?.[0]) {
        expect(page1.items[0].id).not.toBe(page2.items[0].id);
      }
    });

    test("returns empty list when offset exceeds total", async () => {
      const result = await client.Items.List({ limit: 10, offset: 1000 });

      // Should return empty items array (or null)
      expect(result.items?.length ?? 0).toBe(0);
    });

    test("works with default parameters", async () => {
      const result = await client.Items.List();

      expect(result.items).toBeDefined();
      expect(result.total).toBeDefined();
    });
  });

  describe("Items.Get", () => {
    test("returns specific item by ID", async () => {
      const item = await client.Items.Get({ id: 1 });

      expect(item.id).toBe(1);
      expect(item.name).toBeDefined();
      expect(item.price_cents).toBeDefined();
      expect(Array.isArray(item.tags)).toBe(true);
    });

    test("returns not_found for invalid ID", async () => {
      try {
        await client.Items.Get({ id: 999 });
        expect.unreachable("Should have thrown");
      } catch (e) {
        expect(e).toBeInstanceOf(ServerError);
        if (e instanceof ServerError) {
          expect(e.code).toBe("not_found");
        }
      }
    });

    test("returns not_found for negative ID", async () => {
      try {
        await client.Items.Get({ id: -1 });
        expect.unreachable("Should have thrown");
      } catch (e) {
        expect(e).toBeInstanceOf(ServerError);
        if (e instanceof ServerError) {
          expect(e.code).toBe("not_found");
        }
      }
    });
  });

  describe("Items.Create", () => {
    test("creates item with all fields", async () => {
      const item = await client.Items.Create({
        name: "Test Widget",
        description: "A test item",
        price_cents: 1299,
        tags: ["test", "integration"],
      });

      expect(item.id).toBeDefined();
      expect(item.name).toBe("Test Widget");
      expect(item.description).toBe("A test item");
      expect(item.price_cents).toBe(1299);
      expect(item.tags).toEqual(["test", "integration"]);
    });

    test("creates item with minimal fields", async () => {
      const item = await client.Items.Create({
        name: "Minimal Item",
      });

      expect(item.id).toBeDefined();
      expect(item.name).toBe("Minimal Item");
    });

    test("validates required name field", async () => {
      try {
        await client.Items.Create({
          name: "", // Empty name should fail validation
          description: "No name",
          price_cents: 100,
          tags: [],
        });
        expect.unreachable("Should have thrown");
      } catch (e) {
        expect(e).toBeInstanceOf(ServerError);
        if (e instanceof ServerError) {
          expect(e.code).toBe("invalid_argument");
        }
      }
    });

    test("creates items with unique IDs", async () => {
      const item1 = await client.Items.Create({
        name: "Item One",
      });
      const item2 = await client.Items.Create({
        name: "Item Two",
      });

      expect(item1.id).not.toBe(item2.id);
    });
  });
});

describe("Protobuf type handling", () => {
  test("handles optional fields correctly", async () => {
    const item = await client.Items.Create({
      name: "Optional Test",
      // Omitting optional fields
    });

    // Item should be created successfully
    expect(item.id).toBeDefined();
    expect(item.name).toBe("Optional Test");
  });

  test("handles array fields correctly", async () => {
    const item = await client.Items.Create({
      name: "Array Test",
      tags: ["one", "two", "three"],
    });

    expect(item.tags).toEqual(["one", "two", "three"]);
  });

  test("handles empty array fields", async () => {
    const item = await client.Items.Create({
      name: "Empty Array Test",
      tags: [],
    });

    // Empty array or undefined tags
    expect(item.tags?.length ?? 0).toBe(0);
  });

  test("numeric types are handled correctly", async () => {
    const item = await client.Items.Create({
      name: "Numeric Test",
      price_cents: 999999, // Large number to test int64
    });

    expect(item.price_cents).toBe(999999);
  });
});
