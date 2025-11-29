import { describe, expect, it } from "bun:test";
import type { User, Profile, Role, Page } from "./src/types/types";

describe("generated types", () => {
  describe("User and nested types", () => {
    it("accepts valid user with profile", () => {
      const profile: Profile = {
        bio: "Developer",
        links: { github: "https://github.com/test" },
      };

      const user: User = {
        id: "user-1",
        email: "test@example.com",
        role: "admin",
        profile,
        createdAt: "2024-01-01T00:00:00Z",
      };

      expect(user.profile?.bio).toBe("Developer");
    });

    it("accepts user without optional fields", () => {
      const user: User = {
        id: "user-2",
        email: "minimal@example.com",
        role: "viewer",
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
  });

  describe("Generics", () => {
    it("Page<T> works with User", () => {
      const page: Page<User> = {
        items: [
          { id: "1", email: "a@test.com", role: "admin", createdAt: "2024-01-01T00:00:00Z" },
        ],
        total: 100,
        hasMore: true,
      };

      expect(page.items).toHaveLength(1);
      expect(page.total).toBe(100);
    });

    it("Page<T> works with any type", () => {
      const page: Page<string> = {
        items: ["a", "b", "c"],
        total: 3,
        hasMore: false,
      };

      expect(page.items).toHaveLength(3);
    });
  });

  describe("Maps", () => {
    it("Profile.links accepts string map", () => {
      const profile: Profile = {
        bio: "Test",
        links: {
          github: "https://github.com/test",
          twitter: "https://twitter.com/test",
        },
      };

      expect(Object.keys(profile.links!)).toHaveLength(2);
    });

    it("User.metadata accepts any map", () => {
      const user: User = {
        id: "user-1",
        email: "test@example.com",
        role: "admin",
        createdAt: "2024-01-01T00:00:00Z",
        metadata: {
          ip: "127.0.0.1",
          count: 42,
        },
      };

      expect(user.metadata!.ip).toBe("127.0.0.1");
    });
  });
});
