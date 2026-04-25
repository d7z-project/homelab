package workflow

import (
	apiv1 "homelab/pkg/apis/actions/workflow/v1"
	"homelab/pkg/common"
	controllercommon "homelab/pkg/controllers"
	workflowmodel "homelab/pkg/models/workflow"
	workflowservice "homelab/pkg/services/workflow"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ScanWorkflowsHandler godoc
// @Summary Scan all workflows
// @Description Retrieves a list of defined workflow templates with cursor-based pagination.
// @Tags actions
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search"
// @Success 200 {object} common.CursorResponse{items=[]apiv1.Workflow}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 500 {object} common.Response "Internal Server Error"
// @Security ApiKeyAuth
// @Router /actions/workflows [get]
func ScanWorkflowsHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := controllercommon.GetCursorParams(r)
	search := r.URL.Query().Get("search")
	res, err := workflowservice.ScanWorkflows(r.Context(), cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, mapWorkflows(res))
}

// CreateWorkflowHandler godoc
// @Summary Create a workflow
// @Description Creates a new workflow template with the provided steps and configuration.
// @Tags actions
// @Accept json
// @Produce json
// @Param workflow body apiv1.Workflow true "Workflow Configuration"
// @Success 200 {object} apiv1.Workflow
// @Failure 400 {object} common.Response "Bad Request (Validation Error)"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 500 {object} common.Response "Internal Server Error"
// @Security ApiKeyAuth
// @Router /actions/workflows [post]
func CreateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var workflow apiv1.Workflow
	if err := render.Bind(r, &workflow); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	model := toModelWorkflow(workflow)
	res, err := workflowservice.CreateWorkflow(r.Context(), &model)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIWorkflow(*res))
}

// UpdateWorkflowHandler godoc
// @Summary Update a workflow
// @Description Updates an existing workflow template. Performs validation on the new configuration.
// @Tags actions
// @Accept json
// @Produce json
// @Param id path string true "Workflow ID"
// @Param workflow body apiv1.Workflow true "Updated Workflow Configuration"
// @Success 200 {object} apiv1.Workflow
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Workflow Not Found"
// @Security ApiKeyAuth
// @Router /actions/workflows/{id} [put]
func UpdateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var workflow apiv1.Workflow
	if err := render.Bind(r, &workflow); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	model := toModelWorkflow(workflow)
	res, err := workflowservice.UpdateWorkflow(r.Context(), id, &model)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIWorkflow(*res))
}

// DeleteWorkflowHandler godoc
// @Summary Delete a workflow
// @Description Deletes a workflow template and all its associated task instances and triggers.
// @Tags actions
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Workflow Not Found"
// @Security ApiKeyAuth
// @Router /actions/workflows/{id} [delete]
func DeleteWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := workflowservice.DeleteWorkflow(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// GetWorkflowHandler godoc
// @Summary Get a workflow
// @Description Retrieves a specific workflow template by its ID.
// @Tags actions
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {object} apiv1.Workflow
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Workflow Not Found"
// @Security ApiKeyAuth
// @Router /actions/workflows/{id} [get]
func GetWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := workflowservice.GetWorkflow(r.Context(), id)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIWorkflow(*res))
}

// ScanInstancesHandler godoc
// @Summary Scan task instances
// @Description Retrieves historical execution logs with cursor-based pagination.
// @Tags actions
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search"
// @Param workflowId query string false "Workflow ID filter"
// @Success 200 {object} common.CursorResponse{items=[]apiv1.TaskInstance}
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /actions/instances [get]
func ScanInstancesHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := controllercommon.GetCursorParams(r)
	search := r.URL.Query().Get("search")
	workflowId := r.URL.Query().Get("workflowId")
	res, err := workflowservice.ScanTaskInstances(r.Context(), cursor, limit, search, workflowId)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, mapTaskInstances(res))
}

// GetInstanceHandler godoc
// @Summary Get a task instance
// @Description Retrieves a specific task instance by its ID.
// @Tags actions
// @Produce json
// @Param id path string true "Task Instance ID"
// @Success 200 {object} apiv1.TaskInstance
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Instance Not Found"
// @Security ApiKeyAuth
// @Router /actions/instances/{id} [get]
func GetInstanceHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := workflowservice.GetTaskInstance(r.Context(), id)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPITaskInstance(*res))
}

// RunWorkflowHandler godoc
// @Summary Run a workflow manually
// @Description Triggers the immediate execution of a workflow template. Returns the generated instance ID.
// @Tags actions
// @Accept json
// @Produce json
// @Param workflowId path string true "Workflow ID to execute"
// @Param req body apiv1.RunWorkflowRequest false "Workflow Inputs"
// @Success 200 {string} string "instanceId"
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Workflow Not Found"
// @Failure 409 {object} common.Response "Conflict (Already Running)"
// @Security ApiKeyAuth
// @Router /actions/workflows/{workflowId}/run [post]
func RunWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "workflowId")

	var req apiv1.RunWorkflowRequest
	// Ignore errors, body is optional
	_ = render.Bind(r, &req)

	instanceID, err := workflowservice.RunWorkflow(r.Context(), workflowID, req.Inputs, req.Trigger)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, instanceID)
}

// DeleteInstanceHandler godoc
// @Summary Delete a task instance
// @Description Removes a specific task instance and its execution logs.
// @Tags actions
// @Param id path string true "Task Instance ID"
// @Success 200 {string} string "success"
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Instance Not Found"
// @Security ApiKeyAuth
// @Router /actions/instances/{id} [delete]
func DeleteInstanceHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := workflowservice.DeleteTaskInstance(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// CleanupInstancesHandler godoc
// @Summary Cleanup old task instances
// @Description Removes task instances and logs older than the specified number of days.
// @Tags actions
// @Param days query int true "Days older than which instances will be deleted"
// @Success 200 {object} apiv1.TaskCleanupResponse "Number of deleted instances"
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /actions/instances/cleanup [post]
func CleanupInstancesHandler(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		common.BadRequestError(w, r, 0, "invalid days parameter")
		return
	}

	count, err := workflowservice.CleanupTaskInstances(r.Context(), days)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, &apiv1.TaskCleanupResponse{Deleted: count})
}

// GetInstanceLogsHandler godoc
// @Summary Get task instance logs
// @Description Returns execution logs for a specific task instance or step, supporting line offset for real-time refresh.
// @Tags actions
// @Produce json
// @Param id path string true "Task Instance ID"
// @Param stepIndex query int false "Step Index (0 for engine, 1+ for steps)"
// @Param offset query int false "Line offset to start reading from"
// @Success 200 {object} apiv1.TaskLogResponse "Logs and next offset"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 404 {object} common.Response "Instance Not Found"
// @Security ApiKeyAuth
// @Router /actions/instances/{id}/logs [get]
func GetInstanceLogsHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	stepIndexStr := r.URL.Query().Get("stepIndex")
	offsetStr := r.URL.Query().Get("offset")

	if stepIndexStr != "" {
		stepIndex, err := strconv.Atoi(stepIndexStr)
		if err != nil {
			common.BadRequestError(w, r, http.StatusBadRequest, "invalid stepIndex parameter")
			return
		}
		offset := 0
		if offsetStr != "" {
			offset, err = strconv.Atoi(offsetStr)
			if err != nil {
				common.BadRequestError(w, r, http.StatusBadRequest, "invalid offset parameter")
				return
			}
		}

		logs, nextOffset, err := workflowservice.GetStepLogs(r.Context(), id, stepIndex, offset)
		if err != nil {
			controllercommon.HandleError(w, r, err)
			return
		}
		common.Success(w, r, &apiv1.TaskLogResponse{
			Logs:       toAPILogEntries(logs),
			NextOffset: nextOffset,
		})
		return
	}

	// Default to all logs if no stepIndex provided
	logs, err := workflowservice.GetTaskLogs(r.Context(), id)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, &apiv1.TaskLogResponse{
		Logs: toAPILogEntries(logs),
	})
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
	if err := workflowservice.CancelTaskInstance(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// ListManifestsHandler godoc
// @Summary Scan all step manifests
// @Description Returns the specifications (inputs/outputs) for all registered task processors in the system.
// @Tags actions
// @Produce json
// @Success 200 {array} apiv1.StepManifest
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /actions/manifests [get]
func ScanManifestsHandler(w http.ResponseWriter, r *http.Request) {
	res := workflowservice.ScanManifests()
	common.Success(w, r, mapStepManifests(res))
}

// GetWorkflowSchemaHandler godoc
// @Summary Get workflow JSON schema
// @Description Returns the JSON schema for workflow templates, dynamically generated based on available processors.
// @Tags actions
// @Produce json
// @Success 200 {object} apiv1.WorkflowSchemaResponse
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /actions/workflows/schema [get]
func GetWorkflowSchemaHandler(w http.ResponseWriter, r *http.Request) {
	res := workflowservice.GenerateWorkflowSchema()
	common.Success(w, r, &apiv1.WorkflowSchemaResponse{Schema: res})
}

// ProbeHandler godoc
// @Summary Test a single processor
// @Description Executes a specific processor in isolation within a temporary workspace. Useful for debugging or testing parameters.
// @Tags actions
// @Accept json
// @Produce json
// @Param req body apiv1.ProbeRequest true "Probe Configuration"
// @Success 200 {object} apiv1.ProbeResponse "Processor Output Data"
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /actions/probe [post]
func ProbeHandler(w http.ResponseWriter, r *http.Request) {
	var req apiv1.ProbeRequest
	if err := render.Bind(r, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := workflowservice.Probe(r.Context(), req.ProcessorID, req.Params)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, &apiv1.ProbeResponse{
		ProcessorID: req.ProcessorID,
		Outputs:     res,
	})
}

// ValidateWorkflowHandler godoc
// @Summary Validate a workflow configuration
// @Description Checks if a workflow configuration is valid, including variable references and 'if' expressions.
// @Tags actions
// @Accept json
// @Produce json
// @Param workflow body apiv1.Workflow true "Workflow to validate"
// @Success 200 {string} string "success"
// @Failure 400 {object} common.Response "Validation Error"
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /actions/workflows/validate [post]
func ValidateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var workflow apiv1.Workflow
	if err := render.Bind(r, &workflow); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	model := toModelWorkflow(workflow)
	if err := workflowservice.ValidateWorkflow(r.Context(), &model); err != nil {
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
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Workflow Not Found"
// @Security ApiKeyAuth
// @Router /actions/workflows/{id}/webhook/reset [post]
func ResetWebhookTokenHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	token, err := workflowservice.ResetWebhookToken(r.Context(), id)
	if err != nil {
		controllercommon.HandleError(w, r, err)
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

	// Find workflow by token. Since tokens are unique, we scan workflows.
	// We scan with a large limit as a fallback, but ideally we'd have a specific repo method for this.
	// For now, we'll scan through workflows to maintain API consistency.
	res, err := workflowservice.ScanWorkflows(r.Context(), "", 1000, "")
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}

	var target *workflowmodel.Workflow
	for i := range res.Items {
		if res.Items[i].Meta.WebhookEnabled && res.Items[i].Meta.WebhookToken == token {
			target = &res.Items[i]
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
	instanceID, err := workflowservice.TriggerWorkflow(r.Context(), target, target.Meta.ServiceAccountID, "Webhook", inputs)
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
		controllercommon.HandleError(w, r, err)
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
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /actions/validate/regex [post]
func ValidateRegexHandler(w http.ResponseWriter, r *http.Request) {
	regex := r.URL.Query().Get("regex")
	if err := workflowservice.ValidateRegex(regex); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, "success")
}
