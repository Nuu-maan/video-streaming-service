package domain

import "testing"

func TestRoleHasPermission(t *testing.T) {
	allPermissions := []Permission{
		PermissionWatchPublic,
		PermissionWatchPrivate,
		PermissionUploadVideo,
		PermissionDeleteOwnVideo,
		PermissionDeleteAnyVideo,
		PermissionManageUsers,
		PermissionViewAnalytics,
		PermissionModerateContent,
	}

	// granted lists exactly the permissions the role must hold; every other
	// permission in allPermissions must be denied.
	tests := []struct {
		name    string
		role    Role
		granted []Permission
	}{
		{
			name:    "guest",
			role:    RoleGuest,
			granted: []Permission{PermissionWatchPublic},
		},
		{
			name: "user",
			role: RoleUser,
			granted: []Permission{
				PermissionWatchPublic,
				PermissionUploadVideo,
				PermissionDeleteOwnVideo,
			},
		},
		{
			name: "premium",
			role: RolePremium,
			granted: []Permission{
				PermissionWatchPublic,
				PermissionWatchPrivate,
				PermissionUploadVideo,
				PermissionDeleteOwnVideo,
				PermissionViewAnalytics,
			},
		},
		{
			name: "moderator",
			role: RoleModerator,
			granted: []Permission{
				PermissionWatchPublic,
				PermissionWatchPrivate,
				PermissionUploadVideo,
				PermissionDeleteOwnVideo,
				PermissionDeleteAnyVideo,
				PermissionModerateContent,
			},
		},
		{
			name:    "admin",
			role:    RoleAdmin,
			granted: allPermissions,
		},
		{
			name:    "unknown role holds nothing",
			role:    Role("wizard"),
			granted: nil,
		},
		{
			name:    "empty role holds nothing",
			role:    Role(""),
			granted: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grantedSet := make(map[Permission]bool, len(tt.granted))
			for _, p := range tt.granted {
				grantedSet[p] = true
			}

			for _, permission := range allPermissions {
				want := grantedSet[permission]
				if got := tt.role.HasPermission(permission); got != want {
					t.Errorf("Role(%q).HasPermission(%q) = %v, want %v", tt.role, permission, got, want)
				}
			}

			if tt.role.HasPermission(Permission("fly")) {
				t.Errorf("Role(%q).HasPermission(\"fly\") = true, want false for an unknown permission", tt.role)
			}
		})
	}
}

// TestPrivilegedPermissionsAreNotGrantedToUsers pins the privilege boundary that
// matters most: an ordinary user must never be able to delete another person's
// video or administer accounts.
func TestPrivilegedPermissionsAreNotGrantedToUsers(t *testing.T) {
	privileged := []Permission{PermissionDeleteAnyVideo, PermissionManageUsers}

	for _, permission := range privileged {
		if RoleUser.HasPermission(permission) {
			t.Errorf("RoleUser.HasPermission(%q) = true, want false", permission)
		}
		if !RoleAdmin.HasPermission(permission) {
			t.Errorf("RoleAdmin.HasPermission(%q) = false, want true", permission)
		}
	}

	// A moderator may delete any video but may not administer users.
	if !RoleModerator.HasPermission(PermissionDeleteAnyVideo) {
		t.Errorf("RoleModerator.HasPermission(%q) = false, want true", PermissionDeleteAnyVideo)
	}
	if RoleModerator.HasPermission(PermissionManageUsers) {
		t.Errorf("RoleModerator.HasPermission(%q) = true, want false", PermissionManageUsers)
	}
}

func TestRoleIsValid(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want bool
	}{
		{"guest", RoleGuest, true},
		{"user", RoleUser, true},
		{"premium", RolePremium, true},
		{"moderator", RoleModerator, true},
		{"admin", RoleAdmin, true},
		{"empty", Role(""), false},
		{"unknown", Role("wizard"), false},
		{"wrong case", Role("Admin"), false},
		{"superuser is not a role", Role("superuser"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.want {
				t.Errorf("Role(%q).IsValid() = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestRoleString(t *testing.T) {
	if got := RoleAdmin.String(); got != "admin" {
		t.Errorf("RoleAdmin.String() = %q, want %q", got, "admin")
	}
}
