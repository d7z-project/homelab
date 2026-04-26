package rbac

import (
	apiv1 "homelab/pkg/apis/core/rbac/v1"
	discoverymodel "homelab/pkg/models/core/discovery"
	rbacmodel "homelab/pkg/models/core/rbac"
	"homelab/pkg/models/shared"
)

func toModelPolicyRules(items []apiv1.PolicyRule) []rbacmodel.PolicyRule {
	res := make([]rbacmodel.PolicyRule, 0, len(items))
	for _, item := range items {
		res = append(res, rbacmodel.PolicyRule{
			Resource: item.Resource,
			Verbs:    append([]string(nil), item.Verbs...),
		})
	}
	return res
}

func toAPIPolicyRules(items []rbacmodel.PolicyRule) []apiv1.PolicyRule {
	res := make([]apiv1.PolicyRule, 0, len(items))
	for _, item := range items {
		res = append(res, apiv1.PolicyRule{
			Resource: item.Resource,
			Verbs:    append([]string(nil), item.Verbs...),
		})
	}
	return res
}

func toModelServiceAccount(api apiv1.ServiceAccount) rbacmodel.ServiceAccount {
	return rbacmodel.ServiceAccount{
		ID: api.ID,
		Meta: rbacmodel.ServiceAccountV1Meta{
			Name:     api.Meta.Name,
			Comments: api.Meta.Comments,
			Enabled:  api.Meta.Enabled,
		},
		Status: rbacmodel.ServiceAccountV1Status{
			Token:      api.Status.Token,
			LastUsedAt: api.Status.LastUsedAt,
		},
		Generation:      api.Generation,
		ResourceVersion: api.ResourceVersion,
	}
}

func toAPIServiceAccount(model rbacmodel.ServiceAccount) apiv1.ServiceAccount {
	return apiv1.ServiceAccount{
		ID: model.ID,
		Meta: apiv1.ServiceAccountMeta{
			Name:     model.Meta.Name,
			Comments: model.Meta.Comments,
			Enabled:  model.Meta.Enabled,
		},
		Status: apiv1.ServiceAccountStatus{
			Token:      model.Status.Token,
			LastUsedAt: model.Status.LastUsedAt,
		},
		Generation:      model.Generation,
		ResourceVersion: model.ResourceVersion,
	}
}

func toModelRole(api apiv1.Role) rbacmodel.Role {
	return rbacmodel.Role{
		ID: api.ID,
		Meta: rbacmodel.RoleV1Meta{
			Name:     api.Meta.Name,
			Comments: api.Meta.Comments,
			Rules:    toModelPolicyRules(api.Meta.Rules),
		},
		Status:          rbacmodel.RoleV1Status{},
		Generation:      api.Generation,
		ResourceVersion: api.ResourceVersion,
	}
}

func toAPIRole(model rbacmodel.Role) apiv1.Role {
	return apiv1.Role{
		ID: model.ID,
		Meta: apiv1.RoleMeta{
			Name:     model.Meta.Name,
			Comments: model.Meta.Comments,
			Rules:    toAPIPolicyRules(model.Meta.Rules),
		},
		Status:          apiv1.RoleStatus{},
		Generation:      model.Generation,
		ResourceVersion: model.ResourceVersion,
	}
}

func toModelRoleBinding(api apiv1.RoleBinding) rbacmodel.RoleBinding {
	return rbacmodel.RoleBinding{
		ID: api.ID,
		Meta: rbacmodel.RoleBindingV1Meta{
			Name:             api.Meta.Name,
			RoleIDs:          append([]string(nil), api.Meta.RoleIDs...),
			ServiceAccountID: api.Meta.ServiceAccountID,
			Enabled:          api.Meta.Enabled,
		},
		Status:          rbacmodel.RoleBindingV1Status{},
		Generation:      api.Generation,
		ResourceVersion: api.ResourceVersion,
	}
}

func toAPIRoleBinding(model rbacmodel.RoleBinding) apiv1.RoleBinding {
	return apiv1.RoleBinding{
		ID: model.ID,
		Meta: apiv1.RoleBindingMeta{
			Name:             model.Meta.Name,
			RoleIDs:          append([]string(nil), model.Meta.RoleIDs...),
			ServiceAccountID: model.Meta.ServiceAccountID,
			Enabled:          model.Meta.Enabled,
		},
		Status:          apiv1.RoleBindingStatus{},
		Generation:      model.Generation,
		ResourceVersion: model.ResourceVersion,
	}
}

func toAPIResourcePermissions(model *rbacmodel.ResourcePermissions) *apiv1.ResourcePermissions {
	if model == nil {
		return nil
	}
	var rule *apiv1.PolicyRule
	if model.MatchedRule != nil {
		rule = &apiv1.PolicyRule{
			Resource: model.MatchedRule.Resource,
			Verbs:    append([]string(nil), model.MatchedRule.Verbs...),
		}
	}
	return &apiv1.ResourcePermissions{
		AllowedAll:       model.AllowedAll,
		AllowedInstances: append([]string(nil), model.AllowedInstances...),
		MatchedRule:      rule,
	}
}

func mapServiceAccounts(res *shared.PaginationResponse[rbacmodel.ServiceAccount]) *shared.PaginationResponse[apiv1.ServiceAccount] {
	items := make([]apiv1.ServiceAccount, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPIServiceAccount(item))
	}
	return &shared.PaginationResponse[apiv1.ServiceAccount]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}

func mapRoles(res *shared.PaginationResponse[rbacmodel.Role]) *shared.PaginationResponse[apiv1.Role] {
	items := make([]apiv1.Role, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPIRole(item))
	}
	return &shared.PaginationResponse[apiv1.Role]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}

func mapRoleBindings(res *shared.PaginationResponse[rbacmodel.RoleBinding]) *shared.PaginationResponse[apiv1.RoleBinding] {
	items := make([]apiv1.RoleBinding, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPIRoleBinding(item))
	}
	return &shared.PaginationResponse[apiv1.RoleBinding]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}

func mapDiscoverResults(items []discoverymodel.DiscoverResult) []discoverymodel.DiscoverResult {
	return append([]discoverymodel.DiscoverResult(nil), items...)
}
