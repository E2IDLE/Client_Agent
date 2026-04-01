package session

import (
	"Client_Agent/pkg/consts"
	"Client_Agent/pkg/dtos"
	"fmt"
	"log"
	"net"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/pion/stun/v3"
)

type AgentStatus struct {
	AgentName     string      `json:"agentName"`
	AgentVersion  string      `json:"agentVersion"`
	Status        string      `json:"status"`
	Uptime        string      `json:"uptime"`
	MultiAddress  string      `json:"multiAddress"`
	NATType       string      `json:"natType"`
	ConnectedPeer []dtos.Peer `json:"connectedPeer"`
}

type Store struct {
	Status AgentStatus
	Host   host.Host
}

// getPublicAddrViaSTUN은 STUN 서버를 통해 공인 IP와 Port를 가져옵니다.
func getPublicAddrViaSTUN() (ip string, port int, err error) {
	// Google STUN 서버 사용 (변경 가능)
	u, err := stun.ParseURI("stun:stun.l.google.com:19302")
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse STUN URI: %w", err)
	}

	c, err := stun.DialURI(u, &stun.DialConfig{})
	if err != nil {
		return "", 0, fmt.Errorf("failed to connect to STUN server: %w", err)
	}
	defer c.Close()

	var xorAddr stun.XORMappedAddress

	if err = c.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res stun.Event) {
		if res.Error != nil {
			err = fmt.Errorf("STUN request failed: %w", res.Error)
			return
		}
		if getErr := xorAddr.GetFrom(res.Message); getErr != nil {
			err = fmt.Errorf("failed to get XOR-MAPPED-ADDRESS: %w", getErr)
		}
	}); err != nil {
		return "", 0, err
	}

	return xorAddr.IP.String(), xorAddr.Port, nil
}

// buildMultiAddress는 libp2p 호스트의 PeerID와 공인 IP:Port로 multiaddr 문자열을 생성합니다.
func buildMultiAddress(ip string, port int, peerID peer.ID) (string, error) {
	// IPv4 / IPv6 구분
	prefix := "/ip4"
	if net.ParseIP(ip).To4() == nil {
		prefix = "/ip6"
	}

	maStr := fmt.Sprintf("%s/%s/udp/%d/quic-v1/p2p/%s", prefix, ip, port, peerID.String())

	// 유효한 multiaddr인지 검증
	if _, err := multiaddr.NewMultiaddr(maStr); err != nil {
		return "", fmt.Errorf("failed to build multiaddr: %w", err)
	}

	return maStr, nil
}

func initMultiAddress(peerID peer.ID) (string, error) {
	ip, port, err := getPublicAddrViaSTUN()
	if err != nil {
		return "", fmt.Errorf("STUN error: %w", err)
	}

	ma, err := buildMultiAddress(ip, port, peerID)
	if err != nil {
		return "", fmt.Errorf("multiaddr build error: %w", err)
	}

	log.Printf("[INFO] public multiaddr: %s", ma)
	return ma, nil
}

func NewStore() *Store {
	// ── 1. libp2p 호스트 생성 (PeerID 자동 생성) ─────────────────────────────
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/udp/0/quic-v1"),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		log.Printf("[WARN] failed to create libp2p host: %v", err)
	}

	// ── 2. STUN으로 공인 IP:Port 획득 ────────────────────────────────────────
	ip, port, err := getPublicAddrViaSTUN()
	if err != nil {
		log.Printf("[WARN] failed to get public address via STUN: %v", err)
	}

	// ── 3. Multiaddr 생성 (/ip4/IP/tcp/PORT/p2p/PeerID) ──────────────────────
	multiAddress, err := buildMultiAddress(ip, port, h.ID())
	if err != nil {
		log.Printf("[WARN] failed to build multiaddr: %v", err)
	}

	log.Printf("[INFO] public multiaddr: %s", multiAddress)

	return &Store{
		Status: AgentStatus{
			AgentName:     "",
			AgentVersion:  consts.Version,
			Status:        "",
			Uptime:        "",
			MultiAddress:  multiAddress,
			NATType:       "",
			ConnectedPeer: []dtos.Peer{{Nickname: "", Address: "", ConnectionType: "", RTT: 0}},
		},
		Host: h,
	}
}
