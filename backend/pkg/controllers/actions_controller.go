package controllers

import (
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/actions"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ListWorkflowsHandler godoc
// @Summary List all workflows
// @Description Retrieves a list of all defined workflow templates.
// @Tags actions
// @Produce json
// @Success 200 {array} models.Workflow
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 500 {object} common.Response "Internal Server Error"
// @Security ApiKeyAuth
// @Router /actions/workflows [get]
func ListWorkflowsHandler(w http.ResponseWriter, r *http.Request) {
	res, err := actions.ListWorkflows(r.Context())
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, res)
}

// CreateWorkflowHandler godoc
// @Summary Create a workflow
// @Description Creates a new workflow template with the provided steps and configuration.
// @Tags actions
// @Accept json
// @Produce json
// @Param workflow body models.Workflow true "Workflow Configuration"
// @Success 200 {object} models.Workflow
// @Failure 400 {object} common.Response "Bad Request (Validation Error)"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 500 {object} common.Response "Internal Server Error"
// @Security ApiKeyAuth
// @Router /actions/workflows [post]
func CreateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var workflow models.Workflow
	if err := render.Bind(r, &workflow); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := actions.CreateWorkflow(r.Context(), &workflow)
	if err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, res)
}

// UpdateWorkflowHandler godoc
// @Summary Update a workflow
// @Description Updates an existing workflow template. Performs validation on the new configuration.
// @Tags actions
// @Accept json
// @Produce json
// @Param id path string true "Workflow ID"
// @Param workflow body models.Workflow true "Updated Workflow Configuration"
// @Success 200 {object} models.Workflow
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 404 {object} common.Response "Workflow Not Found"
// @Security ApiKeyAuth
// @Router /actions/workflows/{id} [put]
func UpdateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var workflow models.Workflow
	if err := render.Bind(r, &workflow); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := actions.UpdateWorkflow(r.Context(), id, &workflow)
	if err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, res)
}

// DeleteWorkflowHandler godoc
// @Summary Delete a workflow
// @Description Deletes a workflow template and all its associated task instances and triggers.
// @Tags actions
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {string} string "success"
// @Failure 404 {object} common.Response "Workflow Not Found"
// @Security ApiKeyAuth
// @Router /actions/workflows/{id} [delete]
func DeleteWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := actions.DeleteWorkflow(r.Context(), id); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, "success")
}

// ListInstancesHandler godoc
// @Summary List all task instances
// @Description Retrieves a history of all triggered workflow instances and their current status.
// @Tags actions
// @Produce json
// @Success 200 {array} models.TaskInstance
// @Failure 401 {object} common.Response "Unauthorized"
// @Router /actions/instances [get]
func ListInstancesHandler(w http.ResponseWriter, r *http.Request) {
	res, err := actions.ListTaskInstances(r.Context())
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, res)
}

// RunWorkflowHandler godoc
// @Summary Run a workflow manually
// @Description Triggers the immediate execution of a workflow template. Returns the generated instance ID.
// @Tags actions
// @Accept json
// @Produce json
// @Param workflowId path string true "Workflow ID to execute"
// @Param req body models.RunWorkflowRequest false "Workflow Inputs"
// @Success 200 {string} string "instanceId"
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 404 {object} common.Response "Workflow Not Found"
// @Failure 409 {object} common.Response "Conflict (Already Running)"
// @Security ApiKeyAuth
// @Router /actions/workflows/{workflowId}/run [post]
func RunWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "workflowId")

	var req models.RunWorkflowRequest
	// Ignore errors, body is optional
	_ = render.Bind(r, &req)

	instanceID, err := actions.RunWorkflow(r.Context(), workflowID, req.Inputs, req.Trigger)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, instanceID)
}

// DeleteInstanceHandler godoc
// @Summary Delete a task instance
// @Description Removes a specific task instance and its execution logs.
// @Tags actions
// @Param id path string true "Task Instance ID"
// @Success 200 {object} common.Response "success"
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 404 {object} common.Response "Instance Not Found"
// @Security ApiKeyAuth
// @Router /actions/instances/{id} [delete]
func DeleteInstanceHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := actions.DeleteTaskInstance(r.Context(), id); err != nil {
		common.BadRequestError(w, r, 0, err.Error())
		return
	}
	common.Success(w, r, "success")
}

// CleanupInstancesHandler godoc
// @Summary Cleanup old task instances
// @Description Removes task instances and logs older than the specified number of days.
// @Tags actions
// @Param days query int true "Days older than which instances will be deleted"
// @Success 200 {object} map[string]interface{} "Number of deleted instances"
// @Security ApiKeyAuth
// @Router /actions/instances/cleanup [post]
func CleanupInstancesHandler(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		common.BadRequestError(w, r, 0, "invalid days parameter")
		return
	}

	count, err := actions.CleanupTaskInstances(r.Context(), days)
	if err != nil {
		common.InternalServerError(w, r, 0, err.Error())
		return
	}
	common.Success(w, r, map[string]interface{}{"deleted": count})
}

// GetInstanceLogsHandler godoc
// @Summary Get task instance logs
// @Description Returns execution logs for a specific task instance or step, supporting line offset for real-time refresh.
// @Tags actions
// @Produce json
// @Param id path string true "Task Instance ID"
// @Param stepIndex query int false "Step Index (0 for engine, 1+ for steps)"
// @Param offset query int false "Line offset to start reading from"
// @Success 200 {object} map[string]interface{} "Logs and next offset"
// @Failure 404 {object} common.Response "Instance Not Found"
// @Security ApiKeyAuth
// @Router /actions/instances/{id}/logs [get]
func GetInstanceLogsHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	stepIndexStr := r.URL.Query().Get("stepIndex")
	offsetStr := r.URL.Query().Get("offset")

	if stepIndexStr != "" {
		var stepIndex, offset int
		fmt.Sscanf(stepIndexStr, "%d", &stepIndex)
		fmt.Sscanf(offsetStr, "%d", &offset)

		logs, nextOffset, err := actions.GetStepLogs(r.Context(), id, stepIndex, offset)
		if err != nil {
			common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		common.Success(w, r, map[string]interface{}{
			"logs":       logs,
			"nextOffset": nextOffset,
		})
		return
	}

	// Default to all logs if no stepIndex provided
	logs, err := actions.GetTaskLogs(r.Context(), id)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, logs)
}

// CancelInstanceHandler godoc
// @Summary Cancel a task instance
// @Description Attempts to terminate a running task instance gracefully by sending a cancellation signal.
// @Tags actions
// @Produce json
// @Param id path string true "Task Instance ID"
// @Success 200 {string} string "success"
// @Failure 404 {object} common.Response "Instance Not Found"
// @Security ApiKeyAuth
// @Router /actions/instances/{id}/cancel [post]
func CancelInstanceHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := actions.CancelTaskInstance(r.Context(), id); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, "success")
}

// ListManifestsHandler godoc
// @Summary List all step manifests
// @Description Returns the specifications (inputs/outputs) for all registered task processors in the system.
// @Tags actions
// @Produce json
// @Success 200 {array} models.StepManifest
// @Security ApiKeyAuth
// @Router /actions/manifests [get]
func ListManifestsHandler(w http.ResponseWriter, r *http.Request) {
	res := actions.ListManifests()
	common.Success(w, r, res)
}

// GetWorkflowSchemaHandler godoc
// @Summary Get workflow JSON schema
// @Description Returns the JSON schema for workflow templates, dynamically generated based on available processors.
// @Tags actions
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /actions/workflows/schema [get]
func GetWorkflowSchemaHandler(w http.ResponseWriter, r *http.Request) {
	res := actions.GenerateWorkflowSchema()
	common.Success(w, r, res)
}

// ProbeHandler godoc
// @Summary Test a single processor
// @Description Executes a specific processor in isolation within a temporary workspace. Useful for debugging or testing parameters.
// @Tags actions
// @Accept json
// @Produce json
// @Param req body actions.ProbeRequest true "Probe Configuration"
// @Success 200 {object} map[string]string "Processor Output Data"
// @Security ApiKeyAuth
// @Router /actions/probe [post]
func ProbeHandler(w http.ResponseWriter, r *http.Request) {
	var req actions.ProbeRequest
	if err := render.Bind(r, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := actions.Probe(r.Context(), &req)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, res)
}

// ValidateWorkflowHandler godoc
// @Summary Validate a workflow configuration
// @Description Checks if a workflow configuration is valid, including variable references and 'if' expressions.
// @Tags actions
// @Accept json
// @Produce json
// @Param workflow body models.Workflow true "Workflow to validate"
// @Success 200 {string} string "success"
// @Failure 400 {object} common.Response "Validation Error"
// @Security ApiKeyAuth
// @Router /actions/workflows/validate [post]
func ValidateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var workflow models.Workflow
	if err := render.Bind(r, &workflow); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := actions.ValidateWorkflow(r.Context(), &workflow); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, "success")
}

// ResetWebhookTokenHandler godoc
// @Summary Reset a workflow webhook token
// @Description Regenerates the unique token used for Webhook triggering. The old token will be immediately invalidated.
// @Tags actions
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {string} string "New Webhook Token"
// @Failure 404 {object} common.Response "Workflow Not Found"
// @Security ApiKeyAuth
// @Router /actions/workflows/{id}/webhook/reset [post]
func ResetWebhookTokenHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	token, err := actions.ResetWebhookToken(r.Context(), id)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, token)
}

// WebhookHandler godoc
// @Summary Trigger a workflow via webhook
// @Description Asynchronously triggers a workflow using its unique security token. No standard authentication required.
// @Tags actions
// @Produce json
// @Param token path string true "Unique Webhook Token"
// @Success 200 {string} string "instanceId"
// @Failure 401 {object} common.Response "Invalid Token"
// @Failure 403 {object} common.Response "Workflow Disabled"
// @Failure 409 {object} common.Response "Already Running"
// @Router /actions/webhooks/{token} [post]
// @Router /actions/webhooks/{token} [get]
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		common.BadRequestError(w, r, http.StatusBadRequest, "token is required")
		return
	}

	// Find workflow by token
	workflows, err := actions.ListWorkflows(r.Context())
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	var target *models.Workflow
	for _, wf := range workflows {
		if wf.WebhookEnabled && wf.WebhookToken == token {
			target = &wf
			break
		}
	}

	if target == nil {
		common.Error(w, r, http.StatusUnauthorized, http.StatusUnauthorized, "invalid webhook token")
		return
	}

	// Parse query params into inputs map
	inputs := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			inputs[k] = v[0]
		}
	}

	// Asynchronous execution
	instanceID, err := actions.TriggerWorkflow(r.Context(), target, target.ServiceAccountID, "Webhook", inputs)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "is disabled") {
			common.Error(w, r, http.StatusForbidden, http.StatusForbidden, errStr)
			return
		}
		if strings.Contains(errStr, "is already running") {
			common.Error(w, r, http.StatusConflict, http.StatusConflict, errStr)
			return
		}
		common.InternalServerError(w, r, http.StatusInternalServerError, errStr)
		return
	}

	common.Success(w, r, instanceID)
}

// ValidateRegexHandler godoc
// @Summary Validate a regular expression
// @Description Checks if a regex string is syntactically correct for Go.
// @Tags actions
// @Param regex query string true "Regex to validate"
// @Success 200 {string} string "success"
// @Failure 400 {object} common.Response "Invalid Regex"
// @Security ApiKeyAuth
// @Router /actions/validate/regex [post]
func ValidateRegexHandler(w http.ResponseWriter, r *http.Request) {
	regex := r.URL.Query().Get("regex")
	if err := actions.ValidateRegex(regex); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, "success")
}

// ActionsRouter registers the actions routes
func ActionsRouter(r chi.Router) {
	r.Route("/actions", func(r chi.Router) {
		r.Get("/workflows", ListWorkflowsHandler)
		r.Post("/workflows", CreateWorkflowHandler)
		r.Get("/workflows/schema", GetWorkflowSchemaHandler)
		r.Post("/workflows/validate", ValidateWorkflowHandler)
		r.Post("/validate/regex", ValidateRegexHandler)
		r.Put("/workflows/{id}", UpdateWorkflowHandler)
		r.Delete("/workflows/{id}", DeleteWorkflowHandler)
		r.Post("/workflows/{workflowId}/run", RunWorkflowHandler)
		r.Post("/workflows/{id}/webhook/reset", ResetWebhookTokenHandler)

		r.Get("/instances", ListInstancesHandler)
		r.Post("/instances/cleanup", CleanupInstancesHandler)
		r.Get("/instances/{id}/logs", GetInstanceLogsHandler)
		r.Delete("/instances/{id}", DeleteInstanceHandler)
		r.Post("/instances/{id}/cancel", CancelInstanceHandler)

		r.Get("/manifests", ListManifestsHandler)
		r.Post("/probe", ProbeHandler)
	})
}
