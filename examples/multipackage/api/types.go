// Package api provides the main API types that reference both v1 and v2.
package api

import (
	v1 "github.com/broady/tygor/examples/multipackage/api/v1"
	v2 "github.com/broady/tygor/examples/multipackage/api/v2"
)

// MigrationRequest demonstrates referencing same-named types from different packages.
// Without StripPackagePrefix, both User types would collide as "User" in TypeScript.
// With StripPackagePrefix, they become "v1_User" and "v2_User".
type MigrationRequest struct {
	V1User v1.User `json:"v1_user"`
	V2User v2.User `json:"v2_user"`
}

// MigrationResponse returns both user versions.
type MigrationResponse struct {
	Success bool    `json:"success"`
	V1User  v1.User `json:"v1_user"`
	V2User  v2.User `json:"v2_user"`
}
