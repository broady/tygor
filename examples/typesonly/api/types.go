// Package api contains type definitions for standalone TypeScript generation.
package api

import "time"

// [snippet:types]

// User represents a user in the system.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      Role      `json:"role"`
	Profile   *Profile  `json:"profile,omitempty"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"createdAt"`
}

// Profile contains optional user profile information.
type Profile struct {
	Bio       string            `json:"bio"`
	AvatarURL *string           `json:"avatarUrl,omitempty"`
	Links     map[string]string `json:"links"`
	Settings  Settings          `json:"settings"`
}

// Settings holds user preferences.
type Settings struct {
	Theme        Theme  `json:"theme"`
	Locale       string `json:"locale"`
	EmailDigest  bool   `json:"emailDigest"`
	ItemsPerPage int    `json:"itemsPerPage"`
}

// Role represents the access level of a user.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// Theme represents UI color schemes.
type Theme string

const (
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"
	ThemeSystem Theme = "system"
)

// [/snippet:types]

// [snippet:collections]

// Team groups users together.
type Team struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Members []Member `json:"members"`
}

// Member represents a user's membership in a team.
type Member struct {
	UserID   string    `json:"userId"`
	Role     TeamRole  `json:"role"`
	JoinedAt time.Time `json:"joinedAt"`
}

// TeamRole defines permissions within a team.
type TeamRole string

const (
	TeamRoleOwner  TeamRole = "owner"
	TeamRoleMember TeamRole = "member"
	TeamRoleGuest  TeamRole = "guest"
)

// [/snippet:collections]

// [snippet:pagination]

// Pagination contains common pagination parameters.
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

// PageInfo contains pagination metadata for responses.
type PageInfo struct {
	TotalCount  int  `json:"totalCount"`
	HasNextPage bool `json:"hasNextPage"`
	HasPrevPage bool `json:"hasPrevPage"`
}

// PaginatedUsers wraps a user list with pagination info.
type PaginatedUsers struct {
	Users    []User   `json:"users"`
	PageInfo PageInfo `json:"pageInfo"`
}

// [/snippet:pagination]

// [snippet:events]

// EventType categorizes audit events.
type EventType string

const (
	EventTypeUserCreated EventType = "user.created"
	EventTypeUserUpdated EventType = "user.updated"
	EventTypeUserDeleted EventType = "user.deleted"
	EventTypeTeamCreated EventType = "team.created"
)

// AuditEvent records an action in the system.
type AuditEvent struct {
	ID        string         `json:"id"`
	Type      EventType      `json:"type"`
	ActorID   string         `json:"actorId"`
	TargetID  string         `json:"targetId"`
	Metadata  map[string]any `json:"metadata"`
	Timestamp time.Time      `json:"timestamp"`
}

// [/snippet:events]
