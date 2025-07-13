package catalog

import (
	"aegis/sctx"
)

// HasPermission checks if the context includes a specific permission scope
func HasPermission(ctx sctx.SecurityContext, scope string) bool {
	for _, perm := range ctx.Permissions {
		if perm == scope {
			return true
		}
	}
	return false
}

