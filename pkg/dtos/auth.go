package dtos

type StartSessionRequest struct {
	TargetUser string `json:"targetUser"`
}

type StartSessionResponse struct {
	SessionID string `json:"sessionId"`
}
