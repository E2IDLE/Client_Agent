package service

import (
	"directp2p_client_agent/pkg/dtos"
	"directp2p_client_agent/pkg/session"
)

type P2PService struct {
	sessions *session.Store
}

func NewP2PService(sessions *session.Store) *P2PService {
	return &P2PService{sessions: sessions}
}

func (h *P2PService) CreateSession(userAddr string) (dtos.StartSessionResponse, error) {
	token, err := h.sessions.Create(userAddr)
	if err != nil {
		return dtos.StartSessionResponse{}, err
	}

	return dtos.StartSessionResponse{SessionID: token}, nil
}

func (h *P2PService) TryHolePunch(token string) error {
	// TargetUserAddr, ok := h.sessions.Lookup(token)
	// if !ok {
	// 	return
	// }

	return nil
}
