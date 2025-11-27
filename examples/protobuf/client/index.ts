import { createClient, ServerError } from "@tygor/client";
import { registry } from "./src/rpc/manifest";

// [snippet:client-setup]
const client = createClient(registry, {
  baseUrl: "http://localhost:8080",
});
// [/snippet:client-setup]

async function main() {
  try {
    // [snippet:client-calls]
    // Create an item
    const item = await client.Items.Create({
      name: "Widget",
      description: "A useful widget",
      price_cents: 1999,
      tags: ["gadget", "tool"],
    });
    console.log("Created item:", item.id, item.name);

    // List items with pagination
    const list = await client.Items.List({
      limit: 10,
      offset: 0,
    });
    console.log(`Found ${list.total} items:`);
    list.items?.forEach((item) => {
      if (item) {
        console.log(`  - ${item.name}: $${(item.price_cents ?? 0) / 100}`);
      }
    });

    // Get a specific item
    const fetched = await client.Items.Get({ id: item.id });
    console.log("Fetched:", fetched.name, fetched.tags);
    // [/snippet:client-calls]

  } catch (e) {
    if (e instanceof ServerError) {
      console.error(`RPC Error [${e.code}]: ${e.message}`);
    } else {
      throw e;
    }
  }
}

// [snippet:client-errors]
// Error handling example
async function handleErrors() {
  try {
    await client.Items.Get({ id: 99999 });
  } catch (e) {
    if (e instanceof ServerError) {
      console.error(`Error [${e.code}]: ${e.message}`);
    } else {
      throw e;
    }
  }
}
// [/snippet:client-errors]

main();
