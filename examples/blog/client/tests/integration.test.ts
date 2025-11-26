import { describe, test, expect, beforeAll, afterAll } from "bun:test";
import { createClient, RPCError } from "@tygor/client";
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

describe("Users service", () => {
  test("can create a new user", async () => {
    // Username must be alphanumeric (no underscores), 3-20 chars
    const timestamp = Date.now().toString().slice(-8);
    const user = await client.Users.Create({
      username: `test${timestamp}`,
      email: `test${timestamp}@example.com`,
      password: "testpassword123",
    });

    expect(user.id).toBeDefined();
    expect(user.username).toContain("test");
  });

  test("rejects duplicate email", async () => {
    const timestamp = Date.now().toString().slice(-8);
    const email = `dup${timestamp}@example.com`;

    await client.Users.Create({
      username: `usr1${timestamp}`,
      email: email,
      password: "password123",
    });

    try {
      await client.Users.Create({
        username: `usr2${timestamp}`,
        email: email,
        password: "password123",
      });
      expect.unreachable("Should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(RPCError);
      if (e instanceof RPCError) {
        expect(e.code).toBe("invalid_argument");
      }
    }
  });

  test("can login and get token", async () => {
    const result = await client.Users.Login({
      email: "alice@example.com",
      password: "anything", // Demo server accepts any password
    });

    expect(result.token).toBeDefined();
    expect(result.user).toBeDefined();
    expect(result.user?.email).toBe("alice@example.com");
  });

  test("login fails for non-existent user", async () => {
    try {
      await client.Users.Login({
        email: "nonexistent@example.com",
        password: "password",
      });
      expect.unreachable("Should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(RPCError);
      if (e instanceof RPCError) {
        expect(e.code).toBe("unauthenticated");
      }
    }
  });
});

describe("Posts service - public endpoints", () => {
  test("can list published posts without auth", async () => {
    const posts = await client.Posts.List({
      limit: 10,
      offset: 0,
      published: true,
    });

    expect(Array.isArray(posts)).toBe(true);
    // All returned posts should be published
    for (const post of posts) {
      expect(post.published).toBe(true);
    }
  });

  test("can get a specific published post", async () => {
    const post = await client.Posts.Get({ post_id: 1 });

    expect(post.id).toBe(1);
    expect(post.title).toBeDefined();
    expect(post.content).toBeDefined();
  });

  test("returns not_found for non-existent post", async () => {
    try {
      await client.Posts.Get({ post_id: 99999 });
      expect.unreachable("Should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(RPCError);
      if (e instanceof RPCError) {
        expect(e.code).toBe("not_found");
      }
    }
  });

  test("returns permission_denied for unpublished post", async () => {
    try {
      await client.Posts.Get({ post_id: 2 }); // Post 2 is unpublished in demo data
      expect.unreachable("Should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(RPCError);
      if (e instanceof RPCError) {
        expect(e.code).toBe("permission_denied");
      }
    }
  });
});

describe("Posts service - authenticated endpoints", () => {
  let authToken: string;
  let authClient: ReturnType<typeof createClient<typeof registry.manifest>>;

  beforeAll(async () => {
    const login = await client.Users.Login({
      email: "alice@example.com",
      password: "anything",
    });
    authToken = login.token;

    authClient = createClient(registry, {
      baseUrl: server.url,
      headers: () => ({ Authorization: `Bearer ${authToken}` }),
    });
  });

  test("can create post with auth", async () => {
    const post = await authClient.Posts.Create({
      title: "Test Post from Integration Test", // min 5 chars
      content: "This is test content that is longer than 10 chars", // min 10 chars
    });

    expect(post.id).toBeDefined();
    expect(post.title).toBe("Test Post from Integration Test");
    expect(post.published).toBe(false); // New posts start unpublished
  });

  test("can update own post", async () => {
    const created = await authClient.Posts.Create({
      title: "Post to Update Here",
      content: "Original content that is long enough",
    });

    const updated = await authClient.Posts.Update({
      post_id: created.id,
      title: "Updated Title Here",
    });

    expect(updated.title).toBe("Updated Title Here");
    expect(updated.content).toBe("Original content that is long enough"); // Unchanged
  });

  test("can publish own post", async () => {
    const created = await authClient.Posts.Create({
      title: "Post to Publish Here",
      content: "Content to publish that is long enough",
    });

    expect(created.published).toBe(false);

    const published = await authClient.Posts.Publish({
      post_id: created.id,
    });

    expect(published.published).toBe(true);
  });

  test("returns unauthenticated error without token", async () => {
    try {
      await client.Posts.Create({
        title: "Should Fail Title Here", // min 5 chars
        content: "No auth content here", // min 10 chars
      });
      expect.unreachable("Should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(RPCError);
      if (e instanceof RPCError) {
        expect(e.code).toBe("unauthenticated");
      }
    }
  });
});

describe("Comments service", () => {
  let authClient: ReturnType<typeof createClient<typeof registry.manifest>>;

  beforeAll(async () => {
    const login = await client.Users.Login({
      email: "alice@example.com",
      password: "anything",
    });

    authClient = createClient(registry, {
      baseUrl: server.url,
      headers: () => ({ Authorization: `Bearer ${login.token}` }),
    });
  });

  test("can create comment on published post", async () => {
    const comment = await authClient.Comments.Create({
      post_id: 1, // Published post in demo data
      content: "Great post!",
    });

    expect(comment.id).toBeDefined();
    expect(comment.content).toBe("Great post!");
    expect(comment.post_id).toBe(1);
  });

  test("can list comments on a post", async () => {
    const comments = await authClient.Comments.List({
      post_id: 1,
      limit: 10,
      offset: 0,
    });

    expect(Array.isArray(comments)).toBe(true);
    for (const comment of comments) {
      expect(comment.post_id).toBe(1);
    }
  });

  test("cannot comment on unpublished post", async () => {
    try {
      await authClient.Comments.Create({
        post_id: 2, // Unpublished post
        content: "Should fail",
      });
      expect.unreachable("Should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(RPCError);
      if (e instanceof RPCError) {
        expect(e.code).toBe("permission_denied");
      }
    }
  });
});
