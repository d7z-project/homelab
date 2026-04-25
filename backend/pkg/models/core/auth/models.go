package auth

type LoginRequest struct {
	Password string `json:"password"`
	Totp     string `json:"totp"`
}

type LoginResponse struct {
	SessionID string `json:"session_id"`
}

type Session struct {
	ID        string `json:"id"`
	UserType  string `json:"userType"`
	CreatedAt string `json:"createdAt"`
	IP        string `json:"ip"`
	UserAgent string `json:"userAgent"`
}
