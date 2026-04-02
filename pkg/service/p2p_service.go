package service

import (
	"Client_Agent/pkg/session"
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
)

type P2PService struct {
	Sessions *session.Store
}

func NewP2PService(sessions *session.Store) *P2PService {
	return &P2PService{Sessions: sessions}
}

// ─── P2PService ───────────────────────────────────────────────────────────

// TryHolePunch 홀펀칭을 수행하고, 성공하면 Connection을 저장한 뒤
// 고루틴으로 keep-alive를 시작합니다.
func (h *P2PService) TryHolePunch(peerMultiAddr string) error {
	// ── 1. multiaddr 파싱 ──────────────────────────────────────────────────
	ma, err := multiaddr.NewMultiaddr(peerMultiAddr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr %s: %w", peerMultiAddr, err)
	}

	// ── 2. AddrInfo 파싱 ───────────────────────────────────────────────────
	addrInfo, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		return fmt.Errorf("failed to parse AddrInfo: %w", err)
	}

	peerKey := addrInfo.ID.String()

	// ── 3. 기존 커넥션이 살아있으면 재사용 ────────────────────────────────
	h.Sessions.ConnectionsMu.RLock()
	if existing, ok := h.Sessions.Connections[peerKey]; ok {
		h.Sessions.ConnectionsMu.RUnlock()
		// 커넥션이 실제로 살아있는지 확인
		if existing.Conn.IsClosed() {
			// 죽어있으면 아래에서 재연결 (lock 재획득)
			h.Sessions.ConnectionsMu.Lock()
			delete(h.Sessions.Connections, peerKey)
			h.Sessions.ConnectionsMu.Unlock()
		} else {
			return nil // 살아있으면 그대로 반환
		}
	} else {
		h.Sessions.ConnectionsMu.RUnlock()
	}

	// ── 4. peerstore 등록 ─────────────────────────────────────────────────
	h.Sessions.Host.Peerstore().AddAddrs(
		addrInfo.ID, addrInfo.Addrs, peerstore.TempAddrTTL,
	)

	// ── 5. Connect (릴레이 경유 또는 직접) ────────────────────────────────
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connectCancel()

	if err = h.Sessions.Host.Connect(connectCtx, *addrInfo); err != nil {
		return fmt.Errorf("connect to peer %s: %w", addrInfo.ID, err)
	}

	// ── 6. /libp2p/dcutr 스트림 오픈 → 홀펀칭 트리거 ────────────────────
	//      dcutr 핸드셰이크가 끝나면 스트림은 곧바로 닫아도 무방합니다.
	dcutrCtx, dcutrCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dcutrCancel()

	dcutrStream, err := h.Sessions.Host.NewStream(dcutrCtx, addrInfo.ID, "/libp2p/dcutr")
	if err != nil {
		return fmt.Errorf("open dcutr stream to %s: %w", addrInfo.ID, err)
	}
	// dcutr 핸드셰이크가 완료되면 스트림 역할은 끝났으므로 닫습니다.
	defer dcutrStream.Close()

	// ── 7. 홀펀칭 이후 실제 직접 커넥션 획득 ─────────────────────────────
	//      dcutr 이 성공하면 host.Network().ConnsToPeer() 에
	//      직접 경로(non-relay) 커넥션이 추가됩니다.
	directConn, err := h.waitForDirectConn(addrInfo.ID, 10*time.Second)
	if err != nil {
		return fmt.Errorf("direct connection not established after holepunch: %w", err)
	}

	// ── 8. Connection 저장 + keep-alive 고루틴 시작 ───────────────────────
	hpConn := &session.LibP2PConn{
		Conn:   directConn,
		PeerID: addrInfo.ID,
		StopCh: make(chan struct{}),
	}

	h.Sessions.ConnectionsMu.Lock()
	h.Sessions.Connections[peerKey] = hpConn
	h.Sessions.ConnectionsMu.Unlock()

	go h.keepAlive(hpConn)

	return nil
}

// waitForDirectConn dcutr 완료 후 직접 경로(non-relay) 커넥션이
// 생성될 때까지 최대 timeout 동안 폴링합니다.
func (h *P2PService) waitForDirectConn(
	peerID peer.ID, timeout time.Duration,
) (network.Conn, error) {

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, c := range h.Sessions.Host.Network().ConnsToPeer(peerID) {
			if !isRelayConn(c) && !c.IsClosed() {
				return c, nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for direct connection to %s", peerID)
}

// isRelayConn 해당 커넥션이 릴레이(circuit relay)를 경유하는지 판별합니다.
func isRelayConn(c network.Conn) bool {
	for _, proto := range c.RemoteMultiaddr().Protocols() {
		// /p2p-circuit 컴포넌트가 있으면 릴레이 커넥션
		if proto.Code == multiaddr.P_CIRCUIT {
			return true
		}
	}
	return false
}

// keepAlive NAT 테이블 항목 유지를 위해 주기적으로 빈 스트림을 열어
// 패킷을 발생시킵니다. (Stream 재사용 방식보다 안전)
// 가장 짧은 NAT 타임아웃(~20 s)보다 짧은 15 s 간격으로 수행합니다.
func (h *P2PService) keepAlive(conn *session.LibP2PConn) {
	const interval = 15 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := h.sendPing(conn); err != nil {
				// 커넥션이 죽었으면 맵에서 제거 후 종료
				h.CloseConn(conn.PeerID.String())
				return
			}

		case <-conn.StopCh:
			return
		}
	}
}

// sendPing 단기 스트림을 열어 1바이트를 보내고 즉시 닫습니다.
// 커넥션 자체가 살아있는지도 사전 확인합니다.
func (h *P2PService) sendPing(conn *session.LibP2PConn) error {
	if conn.Conn.IsClosed() {
		return fmt.Errorf("connection to %s is closed", conn.PeerID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// /libp2p/keepalive 스트림을 매 틱마다 새로 열고 닫습니다.
	// 열고/닫는 것 자체가 NAT 항목을 갱신하는 패킷을 발생시킵니다.
	s, err := h.Sessions.Host.NewStream(ctx, conn.PeerID, "/libp2p/keepalive")
	if err != nil {
		return fmt.Errorf("open keepalive stream: %w", err)
	}
	defer s.Close()

	_ = s.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = s.Write([]byte{0x0})
	return err
}

// CloseConn 특정 피어의 커넥션을 닫고 keep-alive를 중단합니다.
func (h *P2PService) CloseConn(peerID string) error {
	h.Sessions.ConnectionsMu.Lock()
	defer h.Sessions.ConnectionsMu.Unlock()

	conn, ok := h.Sessions.Connections[peerID]
	if !ok {
		return fmt.Errorf("no holepunch conn found for peer %s", peerID)
	}

	// keep-alive 고루틴 종료 (이미 닫힌 채널이면 패닉 방지)
	select {
	case <-conn.StopCh: // 이미 닫혀 있음
	default:
		close(conn.StopCh)
	}

	// 커넥션 자체를 닫습니다 (내부 모든 스트림 포함)
	err := conn.Conn.Close()
	delete(h.Sessions.Connections, peerID)

	if err != nil {
		return fmt.Errorf("failed to close conn for peer %s: %w", peerID, err)
	}
	return nil
}

// CloseAllConns 모든 커넥션을 닫습니다. (서비스 종료 시)
func (h *P2PService) CloseAllConns() {
	h.Sessions.ConnectionsMu.Lock()
	defer h.Sessions.ConnectionsMu.Unlock()

	for peerID, conn := range h.Sessions.Connections {
		select {
		case <-conn.StopCh:
		default:
			close(conn.StopCh)
		}
		_ = conn.Conn.Close()
		delete(h.Sessions.Connections, peerID)
	}
}
