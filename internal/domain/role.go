package domain

type Role string

const (
	RoleGuest     Role = "guest"
	RoleUser      Role = "user"
	RolePremium   Role = "premium"
	RoleModerator Role = "moderator"
	RoleAdmin     Role = "admin"
)

type Permission string

const (
	PermissionWatchPublic     Permission = "watch_public"
	PermissionWatchPrivate    Permission = "watch_private"
	PermissionUploadVideo     Permission = "upload_video"
	PermissionDeleteOwnVideo  Permission = "delete_own_video"
	PermissionDeleteAnyVideo  Permission = "delete_any_video"
	PermissionManageUsers     Permission = "manage_users"
	PermissionViewAnalytics   Permission = "view_analytics"
	PermissionModerateContent Permission = "moderate_content"
)

var RolePermissions = map[Role][]Permission{
	RoleGuest: {
		PermissionWatchPublic,
	},
	RoleUser: {
		PermissionWatchPublic,
		PermissionUploadVideo,
		PermissionDeleteOwnVideo,
	},
	RolePremium: {
		PermissionWatchPublic,
		PermissionWatchPrivate,
		PermissionUploadVideo,
		PermissionDeleteOwnVideo,
		PermissionViewAnalytics,
	},
	RoleModerator: {
		PermissionWatchPublic,
		PermissionWatchPrivate,
		PermissionUploadVideo,
		PermissionDeleteOwnVideo,
		PermissionDeleteAnyVideo,
		PermissionModerateContent,
	},
	RoleAdmin: {
		PermissionWatchPublic,
		PermissionWatchPrivate,
		PermissionUploadVideo,
		PermissionDeleteOwnVideo,
		PermissionDeleteAnyVideo,
		PermissionManageUsers,
		PermissionViewAnalytics,
		PermissionModerateContent,
	},
}

func (r Role) HasPermission(permission Permission) bool {
	permissions, ok := RolePermissions[r]
	if !ok {
		return false
	}
	
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func (r Role) IsValid() bool {
	validRoles := []Role{RoleGuest, RoleUser, RolePremium, RoleModerator, RoleAdmin}
	for _, valid := range validRoles {
		if r == valid {
			return true
		}
	}
	return false
}

func (r Role) String() string {
	return string(r)
}
