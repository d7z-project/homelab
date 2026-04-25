package routes

import (
	"homelab/pkg/routes/core"

	"github.com/go-chi/chi/v5"
)

func RegisterCore(r chi.Router) {
	core.RegisterDiscovery(r)
	core.RegisterAuth(r)
	core.RegisterSession(r)
	core.RegisterRBAC(r)
	core.RegisterAudit(r)
}
