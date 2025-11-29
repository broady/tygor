import { describe, expect, it } from "bun:test";
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

describe("generated types", () => {
  describe("User and nested types", () => {
    it("accepts valid user with profile", () => {
      const settings: Settings = {
        theme: "dark",
        locale: "en-US",
        emailDigest: true,
        itemsPerPage: 20,
      };

      const profile: Profile = {
        bio: "Developer",
        links: { github: "https://github.com/test" },
        settings,
      };

      const user: User = {
        id: "user-1",
        email: "test@example.com",
        name: "Test User",
        role: "admin",
        tags: ["dev"],
        createdAt: "2024-01-01T00:00:00Z",
        profile,
      };

      expect(user.profile?.settings.theme).toBe("dark");
    });

    it("accepts user without optional profile", () => {
      const user: User = {
        id: "user-2",
        email: "minimal@example.com",
        name: "Minimal",
        role: "viewer",
        tags: [],
        createdAt: "2024-01-01T00:00:00Z",
      };

      expect(user.profile).toBeUndefined();
    });
  });

  describe("String union enums", () => {
    it("Role restricts to valid values", () => {
      const roles: Role[] = ["admin", "editor", "viewer"];
      expect(roles).toHaveLength(3);
    });

    it("Theme restricts to valid values", () => {
      const themes: Theme[] = ["light", "dark", "system"];
      expect(themes).toHaveLength(3);
    });

    it("TeamRole restricts to valid values", () => {
      const teamRoles: TeamRole[] = ["owner", "member", "guest"];
      expect(teamRoles).toHaveLength(3);
    });

    it("EventType restricts to valid values", () => {
      const eventTypes: EventType[] = [
        "user.created",
        "user.updated",
        "user.deleted",
        "team.created",
      ];
      expect(eventTypes).toHaveLength(4);
    });
  });

  describe("Collections", () => {
    it("Team contains array of Members", () => {
      const team: Team = {
        id: "team-1",
        name: "Engineering",
        members: [
          { userId: "user-1", role: "owner", joinedAt: "2024-01-01T00:00:00Z" },
          { userId: "user-2", role: "member", joinedAt: "2024-01-02T00:00:00Z" },
        ],
      };

      expect(team.members).toHaveLength(2);
      expect(team.members![0].role).toBe("owner");
    });
  });

  describe("Maps", () => {
    it("Profile.links accepts string map", () => {
      const profile: Profile = {
        bio: "Test",
        links: {
          github: "https://github.com/test",
          twitter: "https://twitter.com/test",
          website: "https://example.com",
        },
        settings: {
          theme: "light",
          locale: "en",
          emailDigest: false,
          itemsPerPage: 10,
        },
      };

      expect(Object.keys(profile.links!)).toHaveLength(3);
    });

    it("AuditEvent.metadata accepts any map", () => {
      const event: AuditEvent = {
        id: "evt-1",
        type: "user.created",
        actorId: "system",
        targetId: "user-1",
        metadata: {
          ip: "127.0.0.1",
          userAgent: "Mozilla/5.0",
          count: 42,
          nested: { key: "value" },
        },
        timestamp: "2024-01-01T00:00:00Z",
      };

      expect(event.metadata!.ip).toBe("127.0.0.1");
      expect(event.metadata!.count).toBe(42);
    });
  });

  describe("Pagination", () => {
    it("PaginatedUsers wraps users with page info", () => {
      const result: PaginatedUsers = {
        users: [
          {
            id: "user-1",
            email: "a@test.com",
            name: "A",
            role: "viewer",
            tags: [],
            createdAt: "2024-01-01T00:00:00Z",
          },
        ],
        pageInfo: {
          totalCount: 100,
          hasNextPage: true,
          hasPrevPage: false,
        },
      };

      expect(result.pageInfo.totalCount).toBe(100);
      expect(result.pageInfo.hasNextPage).toBe(true);
    });
  });
});
