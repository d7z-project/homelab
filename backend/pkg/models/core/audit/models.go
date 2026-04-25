package audit

type AuditLog struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Subject   string `json:"subject"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	TargetID  string `json:"targetId"`
	Message   string `json:"message"`
	Status    string `json:"status"`
	IPAddress string `json:"ipAddress"`
	UserAgent string `json:"userAgent"`
}

type AuditCleanupResponse struct {
	Deleted int `json:"deleted"`
}
