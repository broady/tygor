import { createClient, ServerError } from "@tygor/client";
import { registry } from "./src/rpc/manifest";

// [snippet:client-setup]
// Create a basic client (for public endpoints)
const client = createClient(registry, {
  baseUrl: "http://localhost:8080",
});
// [/snippet:client-setup]

// [snippet:client-auth]
// Create an authenticated client
function createAuthClient(token: string) {
  return createClient(registry, {
    baseUrl: "http://localhost:8080",
    headers: () => ({
      Authorization: `Bearer ${token}`,
    }),
  });
}
// [/snippet:client-auth]

async function main() {
  try {
    // [snippet:client-login]
    // Login to get a token
    const loginResult = await client.Users.Login({
      email: "alice@example.com",
      password: "anything",
    });
    console.log("Logged in as:", loginResult.user?.username);

    // Create authenticated client with the token
    const authClient = createAuthClient(loginResult.token);
    // [/snippet:client-login]

    // [snippet:client-calls]
    // Public endpoint: list published posts
    const posts = await client.Posts.List({
      limit: 10,
      offset: 0,
      published: true,
    });
    console.log(`Found ${posts.length} published posts`);

    // Authenticated endpoint: create a new post
    const newPost = await authClient.Posts.Create({
      title: "My New Blog Post",
      content: "This is the content of my blog post.",
    });
    console.log("Created post:", newPost.id, newPost.title);

    // Publish the post
    const published = await authClient.Posts.Publish({
      post_id: newPost.id,
    });
    console.log("Published:", published.published);
    // [/snippet:client-calls]

  } catch (e) {
    if (e instanceof ServerError) {
      console.error(`RPC Error [${e.code}]: ${e.message}`);
      if (e.details) {
        console.error("Details:", e.details);
      }
    } else {
      throw e;
    }
  }
}

// [snippet:client-errors]
// Error handling example
async function handleErrors() {
  try {
    await client.Posts.Get({ post_id: 99999 });
  } catch (e) {
    if (e instanceof ServerError) {
      // Structured error from the server
      console.error(`Error [${e.code}]: ${e.message}`);
      // e.details contains validation errors, etc.
    } else {
      throw e; // Re-throw unexpected errors
    }
  }
}
// [/snippet:client-errors]

main();
