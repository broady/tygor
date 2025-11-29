// [snippet:client-usage]

import type { User, Profile, Role, Page } from "./src/types/types";

// Nested types with optional fields
const user: User = {
  id: "user-123",
  email: "alice@example.com",
  role: "admin",
  tags: ["engineering", "leadership"],
  metadata: { source: "api" },
  createdAt: new Date().toISOString(),
  profile: {
    bio: "Software engineer",
    links: { github: "https://github.com/alice" },
  },
};

// String union enums are type-safe
const role: Role = "editor";

// Generics work with any type
const userPage: Page<User> = {
  items: [user],
  total: 1,
  hasMore: false,
};

const stringPage: Page<string> = {
  items: ["a", "b", "c"],
  total: 3,
  hasMore: false,
};

// [/snippet:client-usage]

console.log("User:", user);
console.log("Role:", role);
console.log("User page:", userPage);
console.log("String page:", stringPage);
