package service

import "Client_Agent/pkg/session"

type StreamService struct {
	Sessions *session.Store
}

func NewStreamService(sessions *session.Store) *StreamService {
	return &StreamService{Sessions: sessions}
}
