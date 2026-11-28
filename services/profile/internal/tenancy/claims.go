package tenancy

import (
	"context"

	platformauth "forge.local/platform-auth-go"
	"github.com/google/uuid"

	"profile/internal/identity"
)

// VerifiedClaims returns the claims of the token the platform-auth middleware
// verified for ctx, or nil.
func VerifiedClaims(ctx context.Context) map[string]any {
	if verified, ok := platformauth.IdentityFromContext(ctx); ok {
		return verified.RawClaims
	}
	return nil
}

// verifiedIdentity returns the identity bound to ctx when the platform-auth
// middleware verified its token and it has a tenant.
func verifiedIdentity(ctx context.Context) (identity.IdentityContext, bool) {
	if _, verified := platformauth.IdentityFromContext(ctx); !verified {
		return identity.IdentityContext{}, false
	}
	ident, ok := identity.FromContext(ctx)
	return ident, ok && ident.TenantID != uuid.Nil
}

// unverifiedIdentity returns the identity bound to ctx when no token was
// verified for it (the platform-auth dev mode).
func unverifiedIdentity(ctx context.Context) (identity.IdentityContext, bool) {
	if _, verified := platformauth.IdentityFromContext(ctx); verified {
		return identity.IdentityContext{}, false
	}
	return identity.FromContext(ctx)
}
