import { describe, test, expect, beforeAll, afterAll } from "bun:test";
import { createClient, ServerError } from "@tygor/client";
import { startServer, type RunningServer } from "@tygor/testing";
import { registry } from "../src/rpc/manifest";
import { NewsStatusDraft, NewsStatusPublished, DateTime } from "../src/rpc/types";

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

describe("News service integration", () => {
  describe("News.List", () => {
    test("returns list of news articles", async () => {
      const news = await client.News.List({ limit: 10, offset: 0 });

      expect(Array.isArray(news)).toBe(true);
      expect(news.length).toBeGreaterThan(0);

      // Verify structure of first article
      const article = news[0];
      expect(article).toHaveProperty("id");
      expect(article).toHaveProperty("title");
      expect(article).toHaveProperty("status");
      expect([NewsStatusDraft, NewsStatusPublished, "archived"]).toContain(
        article.status
      );
    });

    test("accepts pagination parameters", async () => {
      // Note: The demo handler doesn't implement pagination,
      // but verifies the parameters are accepted by the server
      const result = await client.News.List({ limit: 1, offset: 0 });

      expect(Array.isArray(result)).toBe(true);
    });

    test("works with empty params", async () => {
      const news = await client.News.List({});

      expect(Array.isArray(news)).toBe(true);
    });
  });

  describe("News.Create", () => {
    test("creates a new article and returns it", async () => {
      const article = await client.News.Create({
        title: "Integration Test Article",
        body: "This is a test article body",
      });

      expect(article.id).toBeDefined();
      expect(article.title).toBe("Integration Test Article");
      expect(article.body).toBe("This is a test article body");
      expect(article.status).toBe(NewsStatusDraft);
      expect(article.created_at).toBeDefined();
    });

    test("creates article without optional body", async () => {
      const article = await client.News.Create({
        title: "Title Only Article",
      });

      expect(article.id).toBeDefined();
      expect(article.title).toBe("Title Only Article");
    });

    test("returns error for invalid input", async () => {
      try {
        await client.News.Create({
          title: "error", // Server simulates error for this title
        });
        expect.unreachable("Should have thrown an error");
      } catch (e) {
        expect(e).toBeInstanceOf(ServerError);
        if (e instanceof ServerError) {
          expect(e.code).toBe("invalid_argument");
          expect(e.message).toBe("simulated error");
        }
      }
    });
  });
});

describe("Type safety verification", () => {
  test("DateTime branded type works correctly", async () => {
    const article = await client.News.Create({
      title: "DateTime Test",
    });

    if (article.created_at) {
      // Verify we can use DateTime helpers
      const date = DateTime.toDate(article.created_at);
      expect(date.getTime()).not.toBeNaN();

      const formatted = DateTime.format(article.created_at);
      expect(typeof formatted).toBe("string");
    }
  });

  test("enum types are properly typed", async () => {
    const news = await client.News.List({});

    for (const article of news) {
      // TypeScript ensures status is one of the enum values
      const validStatuses = [NewsStatusDraft, NewsStatusPublished, "archived"];
      expect(validStatuses).toContain(article.status);
    }
  });
});
