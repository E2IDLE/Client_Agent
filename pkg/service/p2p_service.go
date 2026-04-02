package service

import (
	"Client_Agent/pkg/session"
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

type P2PService struct {
	Sessions *session.Store
}

func NewP2PService(sessions *session.Store) *P2PService {
	return &P2PService{Sessions: sessions}
}

func (h *P2PService) TryHolePunch(peerMultiAddr string) error {
	// ── 1. multiaddr 파싱 ("/ip4/1.2.3.4/tcp/4001/p2p/12D3KooWXXX") ─────────
	ma, err := multiaddr.NewMultiaddr(peerMultiAddr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr %s: %w", peerMultiAddr, err)
	}

	// ── 2. multiaddr → AddrInfo (PeerID + 주소 분리) ─────────────────────────
	addrInfo, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		return fmt.Errorf("failed to parse AddrInfo: %w", err)
	}

	// ── 3. 연결 시도 (timeout 10초) ──────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = h.Sessions.Host.Connect(ctx, *addrInfo); err != nil {
		return fmt.Errorf("failed to connect to peer %s: %w", addrInfo.ID, err)
	}

	return nil
}
