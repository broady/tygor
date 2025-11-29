// [snippet:client-usage]
import { createClient } from "@tygor/client";
import type { Manifest } from "./rpc/manifest";

const client = createClient<Manifest>("http://localhost:8080");

const user = await client.Users.Get({ id: "123" });
// [/snippet:client-usage]

// [snippet:client-call]
const user = await client.Users.Get({ id: 123 });
// user: User (autocomplete works)
// [/snippet:client-call]
