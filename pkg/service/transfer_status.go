package service

import "Client_Agent/pkg/session"

type TransferStatusService struct {
	Sessions *session.Store
}

func NewTransferStatusService(sessions *session.Store) *TransferStatusService {
	return &TransferStatusService{Sessions: sessions}
}
