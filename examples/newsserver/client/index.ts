import { createClient } from '@tygor/client';
import { registry } from './src/rpc/manifest';

// [snippet:client-setup]
// 1. Create the strictly typed client
const client = createClient(
  registry,
  {
    baseUrl: 'http://localhost:8080',
    headers: () => ({
      'Authorization': 'Bearer my-token'
    })
  }
);
// [/snippet:client-setup]

async function main() {
  try {
    console.log("Fetching news...");

    // [snippet:client-calls]
    // 2. Type-safe call: GET /News/List
    const newsList = await client.News.List({
      limit: 10,
      offset: 0
    });
    // [/snippet:client-calls]

    console.log(`Found ${newsList.length} items:`);
    newsList.forEach(item => {
      console.log(`- [${item.id}] ${item.title} (${item.status})`);

      // Demonstrate string timestamp with helpers
      if (item.created_at) {
        const date = new Date(item.created_at);
        console.log(`  Created: ${date.toLocaleString()}`);
      }

      // Demonstrate string union type safety
      if (item.status === "published") {
        console.log(`  ✓ Published`);
      } else if (item.status === "draft") {
        console.log(`  ⚠ Draft`);
      }
    });

    // 3. Type-safe call: POST /News/Create
    console.log("\nCreating news...");
    const newArticle = await client.News.Create({
      title: "Hello Bun",
      body: "Generated from tygor client running in Bun"
    });

    console.log("Created:", newArticle);

    // Demonstrate timestamp handling
    if (newArticle.created_at) {
      const date = new Date(newArticle.created_at);
      console.log(`Created at: ${date.toLocaleString('en-US')}`);
      console.log(`Date object: ${date}`);
    }

    // Demonstrate type safety - timestamps are strings
    // created_at is a string containing an RFC3339 timestamp

  } catch (e: any) {
    console.error("Error:", e.message);
    if (e.details) {
      console.error("Details:", e.details);
    }
  }
}

main();
