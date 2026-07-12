// Package appctx defines the typed keys used to carry request-scoped values
// through context.Context, along with accessors for each.
//
// Keys are an unexported type so they cannot collide with keys set by other
// packages, which is the reason context.WithValue documents against using bare
// strings.
package appctx

import (
	"context"

	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
)

type contextKey int

const (
	keyRequestID contextKey = iota
	keyPrincipal
)

// Principal is the authenticated caller attached to a request. It is absent on
// anonymous requests; callers must handle the not-ok case rather than assuming.
type Principal struct {
	UserID   uuid.UUID
	Username string
	Role     domain.Role
}

// HasPermission reports whether the principal's role grants permission.
func (p Principal) HasPermission(permission domain.Permission) bool {
	return p.Role.HasPermission(permission)
}

// WithRequestID returns a copy of ctx carrying the given request ID.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, keyRequestID, requestID)
}

// RequestID returns the request ID carried by ctx, or "" if there is none.
func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(keyRequestID).(string)
	return requestID
}

// WithPrincipal returns a copy of ctx carrying the authenticated principal.
func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, keyPrincipal, principal)
}

// PrincipalFrom returns the authenticated principal carried by ctx. The bool is
// false for anonymous requests.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(keyPrincipal).(Principal)
	return principal, ok
}
