package routes

import (
	workflowroutes "homelab/pkg/routes/workflow"

	"github.com/go-chi/chi/v5"
)

func RegisterWorkflow(r chi.Router) {
	workflowroutes.Register(r)
}
