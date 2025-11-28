// Example client demonstrating multi-package type disambiguation.
// Notice how v1.User and v2.User become v1_User and v2_User in TypeScript.

// [snippet:client-usage]

import { createClient } from "@tygor/client";
import { registry } from "./src/rpc/manifest";
import type { v1_User, v2_User, MigrationRequest } from "./src/rpc/types";

const client = createClient(registry, { baseUrl: "http://localhost:8080" });

async function main() {
  // Get a v1 user
  const v1User = await client.V1Users.Get({ id: 1 });
  console.log("V1 User:", v1User);

  // Get a v2 user (has additional fields)
  const v2User = await client.V2Users.Get({ id: 1 });
  console.log("V2 User:", v2User);

  // Migrate - both user types are properly typed
  const migrationReq: MigrationRequest = {
    v1_user: { id: 1, name: "Old User" },
    v2_user: { id: 1, name: "New User", email: "new@example.com", created_at: "2024-01-01T00:00:00Z" },
  };

  const result = await client.Migration.Migrate(migrationReq);
  console.log("Migration result:", result);
}

// [/snippet:client-usage]

// Type assertions to verify the types are correct
function typeChecks() {
  // These would fail to compile if types were incorrectly named
  const _v1: v1_User = { id: 1, name: "test" };
  const _v2: v2_User = { id: 1, name: "test", email: "test@test.com", created_at: "2024-01-01" };
  console.log("Type checks passed:", _v1, _v2);
}

typeChecks();
main().catch(console.error);
