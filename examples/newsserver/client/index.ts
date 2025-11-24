import { createClient } from '../../../client/runtime';
import { RPCManifest, RPCMetadata } from './src/rpc/manifest';

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
      console.log(`- [${item.id}] ${item.title}`);
    });

    // 3. Type-safe call: POST /News/Create
    console.log("\nCreating news...");
    const newArticle = await client.News.Create({
      title: "Hello Bun",
      body: "Generated from tygor client running in Bun"
    });
    
    console.log("Created:", newArticle);

  } catch (e: any) {
    console.error("Error:", e.message);
    if (e.details) {
      console.error("Details:", e.details);
    }
  }
}

main();
