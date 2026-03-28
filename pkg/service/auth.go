package service

import (
	"directp2p_client_agent/pkg/session"
)

type AuthService struct {
	sessions *session.Store
}

func NewAuthService(sessions *session.Store) *AuthService {
	return &AuthService{sessions: sessions}
}
