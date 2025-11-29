// [snippet:client-usage]

import type {
  User,
  Profile,
  Settings,
  Role,
  Theme,
  Team,
  Member,
  TeamRole,
  Pagination,
  PageInfo,
  PaginatedUsers,
  AuditEvent,
  EventType,
} from "./src/types/types";

// Nested types with optional fields
const user: User = {
  id: "user-123",
  email: "alice@example.com",
  name: "Alice",
  role: "admin",
  tags: ["engineering", "leadership"],
  createdAt: new Date().toISOString(),
  profile: {
    bio: "Software engineer",
    links: { github: "https://github.com/alice", twitter: "https://twitter.com/alice" },
    settings: {
      theme: "dark",
      locale: "en-US",
      emailDigest: true,
      itemsPerPage: 25,
    },
  },
};

// String union enums are type-safe
const role: Role = "editor";
const theme: Theme = "system";
const teamRole: TeamRole = "owner";

// Arrays of complex nested types
const team: Team = {
  id: "team-456",
  name: "Platform",
  members: [
    { userId: user.id, role: "owner", joinedAt: new Date().toISOString() },
    { userId: "user-789", role: "member", joinedAt: new Date().toISOString() },
  ],
};

// Generic pagination pattern
const paginatedUsers: PaginatedUsers = {
  users: [user],
  pageInfo: { totalCount: 1, hasNextPage: false, hasPrevPage: false },
};

// Event types with metadata maps
const event: AuditEvent = {
  id: "evt-001",
  type: "user.created",
  actorId: "system",
  targetId: user.id,
  metadata: { source: "api", ip: "127.0.0.1" },
  timestamp: new Date().toISOString(),
};

// [/snippet:client-usage]

// [snippet:type-guards]

// Build type guards using the generated union types
function isAdminRole(role: Role): role is "admin" {
  return role === "admin";
}

function isUserEvent(type: EventType): type is "user.created" | "user.updated" | "user.deleted" {
  return type.startsWith("user.");
}

// [/snippet:type-guards]

// Demo
console.log("User:", user);
console.log("Team:", team);
console.log("Event:", event);
console.log("Is admin?", isAdminRole(user.role));
console.log("Is user event?", isUserEvent(event.type));
