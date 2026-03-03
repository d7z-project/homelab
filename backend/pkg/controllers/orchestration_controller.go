package controllers

import (
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/orchestration"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ListWorkflowsHandler godoc
// @Summary List all workflows
// @Tags orchestration
// @Produce json
// @Success 200 {array} models.Workflow
// @Security ApiKeyAuth
// @Router /orchestration/workflows [get]
func ListWorkflowsHandler(w http.ResponseWriter, r *http.Request) {
	res, err := orchestration.ListWorkflows(r.Context())
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, res)
}

// CreateWorkflowHandler godoc
// @Summary Create a workflow
// @Tags orchestration
// @Accept json
// @Produce json
// @Param workflow body models.Workflow true "Workflow"
// @Success 200 {object} models.Workflow
// @Security ApiKeyAuth
// @Router /orchestration/workflows [post]
func CreateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var workflow models.Workflow
	if err := render.Bind(r, &workflow); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := orchestration.CreateWorkflow(r.Context(), &workflow)
	if err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, res)
}

// UpdateWorkflowHandler godoc
// @Summary Update a workflow
// @Tags orchestration
// @Accept json
// @Produce json
// @Param id path string true "Workflow ID"
// @Param workflow body models.Workflow true "Workflow"
// @Success 200 {object} models.Workflow
// @Security ApiKeyAuth
// @Router /orchestration/workflows/{id} [put]
func UpdateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var workflow models.Workflow
	if err := render.Bind(r, &workflow); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := orchestration.UpdateWorkflow(r.Context(), id, &workflow)
	if err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, res)
}

// DeleteWorkflowHandler godoc
// @Summary Delete a workflow
// @Tags orchestration
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {string} string "success"
// @Security ApiKeyAuth
// @Router /orchestration/workflows/{id} [delete]
func DeleteWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := orchestration.DeleteWorkflow(r.Context(), id); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, "success")
}

// ListInstancesHandler godoc
// @Summary List all task instances
// @Tags orchestration
// @Produce json
// @Success 200 {array} models.TaskInstance
// @Security ApiKeyAuth
// @Router /orchestration/instances [get]
func ListInstancesHandler(w http.ResponseWriter, r *http.Request) {
	res, err := orchestration.ListTaskInstances(r.Context())
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, res)
}

// RunWorkflowHandler godoc
// @Summary Run a workflow
// @Tags orchestration
// @Produce json
// @Param workflowId path string true "Workflow ID"
// @Success 200 {string} string "instanceId"
// @Security ApiKeyAuth
// @Router /orchestration/workflows/{workflowId}/run [post]
func RunWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "workflowId")
	instanceID, err := orchestration.RunWorkflow(r.Context(), workflowID)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, instanceID)
}

// GetInstanceLogsHandler godoc
// @Summary Get task instance logs
// @Tags orchestration
// @Produce plain
// @Param id path string true "Instance ID"
// @Success 200 {string} string "logs"
// @Security ApiKeyAuth
// @Router /orchestration/instances/{id}/logs [get]
func GetInstanceLogsHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	logs, err := orchestration.GetTaskLogs(r.Context(), id)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(logs))
}

// CancelInstanceHandler godoc
// @Summary Cancel a task instance
// @Tags orchestration
// @Produce json
// @Param id path string true "Instance ID"
// @Success 200 {string} string "success"
// @Security ApiKeyAuth
// @Router /orchestration/instances/{id}/cancel [post]
func CancelInstanceHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := orchestration.CancelTaskInstance(r.Context(), id); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, "success")
}

// ListManifestsHandler godoc
// @Summary List all step manifests
// @Tags orchestration
// @Produce json
// @Success 200 {array} models.StepManifest
// @Security ApiKeyAuth
// @Router /orchestration/manifests [get]
func ListManifestsHandler(w http.ResponseWriter, r *http.Request) {
	res := orchestration.ListManifests()
	common.Success(w, r, res)
}

// ProbeHandler godoc
// @Summary Probe a processor
// @Tags orchestration
// @Accept json
// @Produce json
// @Param req body orchestration.ProbeRequest true "Probe Request"
// @Success 200 {object} map[string]string
// @Security ApiKeyAuth
// @Router /orchestration/probe [post]
func ProbeHandler(w http.ResponseWriter, r *http.Request) {
	var req orchestration.ProbeRequest
	if err := render.Bind(r, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := orchestration.Probe(r.Context(), &req)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, res)
}

// OrchestrationRouter registers the orchestration routes
func OrchestrationRouter(r chi.Router) {
	r.Route("/orchestration", func(r chi.Router) {
		r.Get("/workflows", ListWorkflowsHandler)
		r.Post("/workflows", CreateWorkflowHandler)
		r.Put("/workflows/{id}", UpdateWorkflowHandler)
		r.Delete("/workflows/{id}", DeleteWorkflowHandler)
		r.Post("/workflows/{workflowId}/run", RunWorkflowHandler)

		r.Get("/instances", ListInstancesHandler)
		r.Get("/instances/{id}/logs", GetInstanceLogsHandler)
		r.Post("/instances/{id}/cancel", CancelInstanceHandler)

		r.Get("/manifests", ListManifestsHandler)
		r.Post("/probe", ProbeHandler)
	})
}
