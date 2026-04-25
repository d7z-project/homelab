package auth

import (
	apiv1 "homelab/pkg/apis/core/auth/v1"
	authmodel "homelab/pkg/models/core/auth"
)

func toAPILoginResponse(sessionID string) *apiv1.LoginResponse {
	return &apiv1.LoginResponse{SessionID: sessionID}
}

func toAPISessions(items []authmodel.Session) []apiv1.Session {
	res := make([]apiv1.Session, 0, len(items))
	for _, item := range items {
		res = append(res, apiv1.Session{
			ID:        item.ID,
			UserType:  item.UserType,
			CreatedAt: item.CreatedAt,
			IP:        item.IP,
			UserAgent: item.UserAgent,
		})
	}
	return res
}
