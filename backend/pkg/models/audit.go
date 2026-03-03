package models

import "net/http"

type AuditLog struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"` // RFC3339 format
	Subject   string `json:"subject"`   // User or SA name (e.g., "root" or "sa-name")
	Action    string `json:"action"`    // CREATE, UPDATE, DELETE, or HTTP Method
	Resource  string `json:"resource"`  // e.g., "DNS/Record", "RBAC/ServiceAccount"
	TargetID  string `json:"targetId"`  // ID or name of the resource operated on
	Message   string `json:"message"`   // Detailed description
	Status    string `json:"status"`    // Success or Failed
	IPAddress string `json:"ipAddress"`
	UserAgent string `json:"userAgent"`
}

func (a *AuditLog) Bind(r *http.Request) error {
	return nil
}

type AuditCleanupResponse struct {
	Deleted int `json:"deleted"`
}

func (a *AuditCleanupResponse) Bind(r *http.Request) error {
	return nil
}
