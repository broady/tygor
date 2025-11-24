import { createClient } from '@tygor/client';
import { RPCManifest, RPCMetadata } from './src/rpc/manifest';
import { DateTime, NewsStatusPublished, NewsStatusDraft } from './src/rpc/types';

// 1. Create the strictly typed client
const client = createClient<RPCManifest>(
  {
    baseUrl: 'http://localhost:8080',
    headers: () => ({
      'Authorization': 'Bearer my-token'
    })
  },
  RPCMetadata
);

async function main() {
  try {
    console.log("Fetching news...");

    // 2. Type-safe call: GET /News/List
    const newsList = await client.News.List({
      limit: 10,
      offset: 0
    });

    console.log(`Found ${newsList.length} items:`);
    newsList.forEach(item => {
      console.log(`- [${item.id}] ${item.title} (${item.status})`);

      // Demonstrate DateTime branded type with helpers
      if (item.created_at) {
        console.log(`  Created: ${DateTime.format(item.created_at)}`);
      }

      // Demonstrate enum type safety
      if (item.status === NewsStatusPublished) {
        console.log(`  ✓ Published`);
      } else if (item.status === NewsStatusDraft) {
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

    // Demonstrate DateTime helpers
    if (newArticle.created_at) {
      console.log(`Created at: ${DateTime.format(newArticle.created_at, 'en-US')}`);
      console.log(`Date object: ${DateTime.toDate(newArticle.created_at)}`);
    }

    // Demonstrate type safety - these would be TypeScript errors:
    // const str: string = newArticle.created_at; // ❌ Error: DateTime is not assignable to string
    // newArticle.created_at = "2024-01-01"; // ❌ Error: string is not assignable to DateTime
    // newArticle.created_at = DateTime.from("2024-01-01T00:00:00Z"); // ✅ OK

  } catch (e: any) {
    console.error("Error:", e.message);
    if (e.details) {
      console.error("Details:", e.details);
    }
  }
}

main();
