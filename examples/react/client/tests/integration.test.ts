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

describe("Tasks service integration", () => {
  describe("Tasks.List", () => {
    test("returns empty list initially", async () => {
      const tasks = await client.Tasks.List({});

      expect(Array.isArray(tasks)).toBe(true);
    });

    test("accepts show_done parameter", async () => {
      const tasks = await client.Tasks.List({ show_done: false });

      expect(Array.isArray(tasks)).toBe(true);
    });
  });

  describe("Tasks.Create", () => {
    test("creates a new task and returns it", async () => {
      const task = await client.Tasks.Create({
        title: "Test task",
      });

      expect(task.id).toBeDefined();
      expect(task.title).toBe("Test task");
      expect(task.done).toBe(false);
    });
  });

  describe("Tasks.Toggle", () => {
    test("toggles task done status", async () => {
      // Create a task first
      const created = await client.Tasks.Create({
        title: "Toggle test",
      });
      expect(created.done).toBe(false);

      // Toggle it
      const toggled = await client.Tasks.Toggle({ id: created.id });
      expect(toggled.done).toBe(true);

      // Toggle again
      const toggledBack = await client.Tasks.Toggle({ id: created.id });
      expect(toggledBack.done).toBe(false);
    });

    test("returns error for non-existent task", async () => {
      try {
        await client.Tasks.Toggle({ id: 99999 });
        expect.unreachable("Should have thrown an error");
      } catch (e) {
        expect(e).toBeInstanceOf(ServerError);
        if (e instanceof ServerError) {
          expect(e.code).toBe("not_found");
          expect(e.message).toBe("task not found");
        }
      }
    });
  });
});

describe("Full workflow", () => {
  test("create, list, toggle, list", async () => {
    // Create a task
    const task = await client.Tasks.Create({ title: "Workflow test" });

    // List should include it
    let tasks = await client.Tasks.List({});
    expect(tasks.some((t) => t.id === task.id)).toBe(true);

    // Toggle to done
    await client.Tasks.Toggle({ id: task.id });

    // List with show_done=false should exclude it
    tasks = await client.Tasks.List({ show_done: false });
    expect(tasks.some((t) => t.id === task.id)).toBe(false);

    // List with show_done=true should include it
    tasks = await client.Tasks.List({ show_done: true });
    expect(tasks.some((t) => t.id === task.id)).toBe(true);
  });
});
