package workflow

import (
	apiv1 "homelab/pkg/apis/actions/workflow/v1"
	"homelab/pkg/models/shared"
	workflowmodel "homelab/pkg/models/workflow"
)

func toModelVarDefinitions(items map[string]apiv1.VarDefinition) map[string]workflowmodel.VarDefinition {
	if items == nil {
		return nil
	}
	res := make(map[string]workflowmodel.VarDefinition, len(items))
	for key, item := range items {
		res[key] = workflowmodel.VarDefinition{
			Description:   item.Description,
			Default:       item.Default,
			Required:      item.Required,
			RegexFrontend: item.RegexFrontend,
			RegexBackend:  item.RegexBackend,
		}
	}
	return res
}

func toAPIVarDefinitions(items map[string]workflowmodel.VarDefinition) map[string]apiv1.VarDefinition {
	if items == nil {
		return nil
	}
	res := make(map[string]apiv1.VarDefinition, len(items))
	for key, item := range items {
		res[key] = apiv1.VarDefinition{
			Description:   item.Description,
			Default:       item.Default,
			Required:      item.Required,
			RegexFrontend: item.RegexFrontend,
			RegexBackend:  item.RegexBackend,
		}
	}
	return res
}

func toModelSteps(items []apiv1.Step) []workflowmodel.Step {
	res := make([]workflowmodel.Step, 0, len(items))
	for _, item := range items {
		res = append(res, workflowmodel.Step{
			ID:     item.ID,
			Type:   item.Type,
			Name:   item.Name,
			If:     item.If,
			Params: cloneStringMap(item.Params),
			Fail:   item.Fail,
		})
	}
	return res
}

func toAPISteps(items []workflowmodel.Step) []apiv1.Step {
	res := make([]apiv1.Step, 0, len(items))
	for _, item := range items {
		res = append(res, apiv1.Step{
			ID:     item.ID,
			Type:   item.Type,
			Name:   item.Name,
			If:     item.If,
			Params: cloneStringMap(item.Params),
			Fail:   item.Fail,
		})
	}
	return res
}

func toModelWorkflow(api apiv1.Workflow) workflowmodel.Workflow {
	return workflowmodel.Workflow{
		ID: api.ID,
		Meta: workflowmodel.WorkflowV1Meta{
			Name:             api.Meta.Name,
			Description:      api.Meta.Description,
			Enabled:          api.Meta.Enabled,
			Timeout:          api.Meta.Timeout,
			ServiceAccountID: api.Meta.ServiceAccountID,
			CronEnabled:      api.Meta.CronEnabled,
			CronExpr:         api.Meta.CronExpr,
			WebhookEnabled:   api.Meta.WebhookEnabled,
			WebhookToken:     api.Meta.WebhookToken,
			Vars:             toModelVarDefinitions(api.Meta.Vars),
			Steps:            toModelSteps(api.Meta.Steps),
		},
		Status: workflowmodel.WorkflowV1Status{
			CreatedAt: api.Status.CreatedAt,
			UpdatedAt: api.Status.UpdatedAt,
		},
		Generation:      api.Generation,
		ResourceVersion: api.ResourceVersion,
	}
}

func toAPIWorkflow(model workflowmodel.Workflow) apiv1.Workflow {
	return apiv1.Workflow{
		ID: model.ID,
		Meta: apiv1.WorkflowMeta{
			Name:             model.Meta.Name,
			Description:      model.Meta.Description,
			Enabled:          model.Meta.Enabled,
			Timeout:          model.Meta.Timeout,
			ServiceAccountID: model.Meta.ServiceAccountID,
			CronEnabled:      model.Meta.CronEnabled,
			CronExpr:         model.Meta.CronExpr,
			WebhookEnabled:   model.Meta.WebhookEnabled,
			WebhookToken:     model.Meta.WebhookToken,
			Vars:             toAPIVarDefinitions(model.Meta.Vars),
			Steps:            toAPISteps(model.Meta.Steps),
		},
		Status: apiv1.WorkflowStatus{
			CreatedAt: model.Status.CreatedAt,
			UpdatedAt: model.Status.UpdatedAt,
		},
		Generation:      model.Generation,
		ResourceVersion: model.ResourceVersion,
	}
}

func toAPILogEntries(items []workflowmodel.LogEntry) []apiv1.LogEntry {
	res := make([]apiv1.LogEntry, 0, len(items))
	for _, item := range items {
		res = append(res, apiv1.LogEntry{
			Timestamp: item.Timestamp,
			StepID:    item.StepID,
			Message:   item.Message,
		})
	}
	return res
}

func toAPIStepTimings(items map[int]*workflowmodel.StepTiming) map[int]*apiv1.StepTiming {
	if items == nil {
		return nil
	}
	res := make(map[int]*apiv1.StepTiming, len(items))
	for key, item := range items {
		if item == nil {
			res[key] = nil
			continue
		}
		res[key] = &apiv1.StepTiming{
			StartedAt:  item.StartedAt,
			FinishedAt: item.FinishedAt,
		}
	}
	return res
}

func toAPITaskInstance(model workflowmodel.TaskInstance) apiv1.TaskInstance {
	return apiv1.TaskInstance{
		ID: model.ID,
		Meta: apiv1.TaskInstanceMeta{
			WorkflowID:       model.Meta.WorkflowID,
			Trigger:          model.Meta.Trigger,
			UserID:           model.Meta.UserID,
			ServiceAccountID: model.Meta.ServiceAccountID,
			Inputs:           cloneStringMap(model.Meta.Inputs),
			Workspace:        model.Meta.Workspace,
			Steps:            toAPISteps(model.Meta.Steps),
		},
		Status: apiv1.TaskInstanceStatus{
			Status:      string(model.Status.Status),
			CurrentStep: model.Status.CurrentStep,
			StartedAt:   model.Status.StartedAt,
			FinishedAt:  model.Status.FinishedAt,
			Error:       model.Status.Error,
			Outputs:     cloneStringMap(model.Status.Outputs),
			Logs:        toAPILogEntries(model.Status.Logs),
			StepTimings: toAPIStepTimings(model.Status.StepTimings),
		},
		Generation:      model.Generation,
		ResourceVersion: model.ResourceVersion,
	}
}

func toAPIStepManifest(model workflowmodel.StepManifest) apiv1.StepManifest {
	params := make([]apiv1.ParamDefinition, 0, len(model.Params))
	for _, item := range model.Params {
		params = append(params, apiv1.ParamDefinition{
			Name:          item.Name,
			Description:   item.Description,
			Optional:      item.Optional,
			RegexFrontend: item.RegexFrontend,
			RegexBackend:  item.RegexBackend,
			LookupCode:    item.LookupCode,
		})
	}
	outputs := make([]apiv1.ParamDefinition, 0, len(model.OutputParams))
	for _, item := range model.OutputParams {
		outputs = append(outputs, apiv1.ParamDefinition{
			Name:          item.Name,
			Description:   item.Description,
			Optional:      item.Optional,
			RegexFrontend: item.RegexFrontend,
			RegexBackend:  item.RegexBackend,
			LookupCode:    item.LookupCode,
		})
	}
	return apiv1.StepManifest{
		ID:           model.ID,
		Name:         model.Name,
		Description:  model.Description,
		Params:       params,
		OutputParams: outputs,
	}
}

func mapWorkflows(res *shared.PaginationResponse[workflowmodel.Workflow]) *shared.PaginationResponse[apiv1.Workflow] {
	items := make([]apiv1.Workflow, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPIWorkflow(item))
	}
	return &shared.PaginationResponse[apiv1.Workflow]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}

func mapTaskInstances(res *shared.PaginationResponse[workflowmodel.TaskInstance]) *shared.PaginationResponse[apiv1.TaskInstance] {
	items := make([]apiv1.TaskInstance, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPITaskInstance(item))
	}
	return &shared.PaginationResponse[apiv1.TaskInstance]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}

func mapStepManifests(items []workflowmodel.StepManifest) []apiv1.StepManifest {
	res := make([]apiv1.StepManifest, 0, len(items))
	for _, item := range items {
		res = append(res, toAPIStepManifest(item))
	}
	return res
}

func cloneStringMap(items map[string]string) map[string]string {
	if items == nil {
		return nil
	}
	res := make(map[string]string, len(items))
	for key, value := range items {
		res[key] = value
	}
	return res
}
