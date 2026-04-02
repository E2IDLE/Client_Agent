package service

import (
	"Client_Agent/pkg/dtos"
	"Client_Agent/pkg/session"
)

type StatusService struct {
	Sessions *session.Store
}

func NewStatusService(sessions *session.Store) *StatusService {
	return &StatusService{Sessions: sessions}
}

func (h *StatusService) GetStatus() dtos.AgentStatusResponse {
	return dtos.AgentStatusResponse{
		AgentName:     h.Sessions.Status.AgentName,
		AgentVersion:  h.Sessions.Status.AgentVersion,
		Status:        h.Sessions.Status.Status,
		Uptime:        h.Sessions.Status.Uptime,
		PeerID:        h.Sessions.Status.PeerID,
		MultiAddress:  h.Sessions.Status.MultiAddress,
		NATType:       h.Sessions.Status.NATType,
		ConnectedPeer: h.Sessions.Status.ConnectedPeer,
	}
}
