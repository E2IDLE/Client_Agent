package session

import (
	"Client_Agent/pkg/consts"
	"Client_Agent/pkg/dtos"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/pion/stun/v3"
)

// ── 전송 상태 상수 ────────────────────────────────────────────────────────
type TransferStatus string

const (
	TransferStatusPending   TransferStatus = "pending"
	TransferStatusSending   TransferStatus = "sending"
	TransferStatusDone      TransferStatus = "done"
	TransferStatusFailed    TransferStatus = "failed"
	TransferStatusCancelled TransferStatus = "cancelled"
)

type AgentStatus struct {
	AgentName     string      `json:"agentName"`
	AgentVersion  string      `json:"agentVersion"`
	Status        string      `json:"status"`
	Uptime        string      `json:"uptime"`
	PeerID        string      `json:"peerId"`
	MultiAddress  string      `json:"multiAddress"`
	NATType       string      `json:"natType"`
	ConnectedPeer []dtos.Peer `json:"connectedPeer"`
}

// ── CountingStream ────────────────────────────────────────────────────────
type CountingStream struct {
	network.Stream
	bytesSent     int64
	bytesReceived int64
}

func (s *CountingStream) Write(p []byte) (int, error) {
	n, err := s.Stream.Write(p)
	atomic.AddInt64(&s.bytesSent, int64(n))
	return n, err
}

func (s *CountingStream) Read(p []byte) (int, error) {
	n, err := s.Stream.Read(p)
	atomic.AddInt64(&s.bytesReceived, int64(n))
	return n, err
}

// 외부에서 안전하게 읽을 수 있도록 getter 제공
func (s *CountingStream) BytesSent() int64 {
	return atomic.LoadInt64(&s.bytesSent)
}

func (s *CountingStream) BytesReceived() int64 {
	return atomic.LoadInt64(&s.bytesReceived)
}

const speedHistorySize = 10

type ByteSnapshot struct {
	Bytes     int64
	Timestamp time.Time
}

// ── SendingFile ───────────────────────────────────────────────────────────
type SendingFile struct {
	TransferId string
	PeerID     string
	Status     TransferStatus
	TotalFiles int
	SentFiles  int // 완료된 파일 수
	Files      []dtos.FileInfo
	Stream     *CountingStream

	// 속도 추적
	StartedAt    time.Time
	LastSnapshot ByteSnapshot
	SpeedHistory []int64
	SpeedMu      sync.Mutex

	// 속도 제한 (바이트/초, 0 = 무제한)
	SpeedLimit int64
}

type ReceivingFile struct {
	TransferId    string
	PeerID        string
	Status        TransferStatus
	TotalFiles    int
	ReceivedFiles int
	Files         []dtos.FileInfo
	Stream        *CountingStream

	// 속도 추적
	StartedAt    time.Time
	LastSnapshot ByteSnapshot
	SpeedHistory []int64
	SpeedMu      sync.Mutex
}

type LibP2PConn struct {
	Conn   network.Conn // libp2p 트랜스포트 커넥션
	PeerID peer.ID
	StopCh chan struct{}
}

// ── Store ─────────────────────────────────────────────────────────────────
type Store struct {
	Status AgentStatus
	Host   host.Host

	// key: transferId
	SendingFiles   map[string]*SendingFile
	SendingFilesMu sync.RWMutex

	ReceivingFiles   map[string]*ReceivingFile
	ReceivingFilesMu sync.RWMutex

	// key: peer.ID.String()
	Connections   map[string]*LibP2PConn
	ConnectionsMu sync.RWMutex
}

// ── Store 메서드 ──────────────────────────────────────────────────────────

func (s *Store) SetSendingFile(transferId string, sf SendingFile) {
	s.SendingFilesMu.Lock()
	defer s.SendingFilesMu.Unlock()
	s.SendingFiles[transferId] = &sf
}

func (s *Store) UpdateSendingFileStatus(transferId string, status TransferStatus) {
	s.SendingFilesMu.Lock()
	defer s.SendingFilesMu.Unlock()
	if sf, ok := s.SendingFiles[transferId]; ok {
		sf.Status = status
	}
}

func (s *Store) UpdateSendingFileProgress(transferId string, sentFiles int) {
	s.SendingFilesMu.Lock()
	defer s.SendingFilesMu.Unlock()
	if sf, ok := s.SendingFiles[transferId]; ok {
		sf.SentFiles = sentFiles
	}
}

// 전송 현황 조회 (ex. GET /transfers/:id 등에서 활용)
func (s *Store) GetSendingFile(transferId string) (*SendingFile, bool) {
	s.SendingFilesMu.RLock()
	defer s.SendingFilesMu.RUnlock()
	sf, ok := s.SendingFiles[transferId]
	return sf, ok
}

func (s *Store) SetReceivingFile(transferId string, rf ReceivingFile) {
	s.ReceivingFilesMu.Lock()
	defer s.ReceivingFilesMu.Unlock()
	s.ReceivingFiles[transferId] = &rf
}

func (s *Store) UpdateReceivingFileStatus(transferId string, status TransferStatus) {
	s.ReceivingFilesMu.Lock()
	defer s.ReceivingFilesMu.Unlock()
	if rf, ok := s.ReceivingFiles[transferId]; ok {
		rf.Status = status
	}
}

func (s *Store) UpdateReceivingFileProgress(transferId string, receivedFiles int) {
	s.ReceivingFilesMu.Lock()
	defer s.ReceivingFilesMu.Unlock()
	if rf, ok := s.ReceivingFiles[transferId]; ok {
		rf.ReceivedFiles = receivedFiles
	}
}

func (s *Store) GetReceivingFile(transferId string) (*ReceivingFile, bool) {
	s.ReceivingFilesMu.RLock()
	defer s.ReceivingFilesMu.RUnlock()
	rf, ok := s.ReceivingFiles[transferId]
	return rf, ok
}

func (s *Store) handleIncomingFileStream(stream network.Stream) {
	// CountingStream으로 래핑
	cs := &CountingStream{Stream: stream}
	remotePeerID := stream.Conn().RemotePeer().String()

	go func() {
		defer stream.Close()

		// ── Step 1. TransferHeader 수신 ──────────────────────────────────
		headerBytes, err := readWithLength(cs)
		if err != nil {
			fmt.Printf("[recv] failed to read header from %s: %v\n", remotePeerID, err)
			return
		}

		type TransferHeader struct {
			TransferId string          `json:"transferId"`
			Files      []dtos.FileInfo `json:"files"`
		}
		var header TransferHeader
		if err := json.Unmarshal(headerBytes, &header); err != nil {
			fmt.Printf("[recv] failed to unmarshal header: %v\n", err)
			return
		}

		// ── Step 2. Store에 수신 정보 등록 ──────────────────────────────
		rf := ReceivingFile{
			TransferId: header.TransferId,
			PeerID:     remotePeerID,
			Status:     TransferStatusPending,
			TotalFiles: len(header.Files),
			Files:      header.Files,
			Stream:     cs,
		}
		s.SetReceivingFile(header.TransferId, rf)
		s.UpdateReceivingFileStatus(header.TransferId, TransferStatusSending)

		// ── Step 3. 파일 저장 경로 준비 ─────────────────────────────────
		saveDir := filepath.Join(consts.FileSaveDir, header.TransferId)
		if err := os.MkdirAll(saveDir, 0755); err != nil {
			fmt.Printf("[recv] failed to create save dir: %v\n", err)
			s.UpdateReceivingFileStatus(header.TransferId, TransferStatusFailed)
			return
		}

		// ── Step 4. 파일 순차 수신 ──────────────────────────────────────
		for i := range header.Files {
			if err := receiveFile(cs, saveDir); err != nil {
				fmt.Printf("[recv] failed to receive file %d: %v\n", i+1, err)
				s.UpdateReceivingFileStatus(header.TransferId, TransferStatusFailed)
				return
			}
			s.UpdateReceivingFileProgress(header.TransferId, i+1)
		}

		// ── Step 5. 완료 ─────────────────────────────────────────────────
		s.UpdateReceivingFileStatus(header.TransferId, TransferStatusDone)
		fmt.Printf("[recv] transfer %s done — received %d file(s) from %s\n",
			header.TransferId, len(header.Files), remotePeerID)
	}()
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
	peerId := h.ID()
	multiAddress, err := buildMultiAddress(ip, port, peerId)
	if err != nil {
		log.Printf("[WARN] failed to build multiaddr: %v", err)
	}

	log.Printf("[INFO] public multiaddr: %s", multiAddress)

	// 스트림 핸들러 등록
	h.SetStreamHandler("/libp2p/keepalive", func(s network.Stream) {
		go func() {
			defer s.Reset()
			buf := make([]byte, 1)
			for {
				if _, err := s.Read(buf); err != nil {
					return // 연결 끊기면 자동 종료
				}
			}
		}()
	})

	s := Store{
		Status: AgentStatus{
			AgentName:     "",
			AgentVersion:  consts.Version,
			Status:        "",
			Uptime:        "",
			PeerID:        peerId.String(),
			MultiAddress:  multiAddress,
			NATType:       "",
			ConnectedPeer: []dtos.Peer{},
		},
		Host:           h,
		SendingFiles:   make(map[string]*SendingFile),
		ReceivingFiles: make(map[string]*ReceivingFile),
		Connections:    make(map[string]*LibP2PConn),
	}

	// 파일 스트림 핸들러 등록
	h.SetStreamHandler(consts.FileTransferProtocol, s.handleIncomingFileStream)

	return &s
}

// ── receiveFile ───────────────────────────────────────────────────────────
//
// 프로토콜 (송신 측과 대칭):
//   [4B]  metaLen   — FileInfo JSON 길이
//   [NB]  meta      — FileInfo JSON
//   [8B]  fileSize  — 파일 바이트 크기
//   [MB]  fileData  — 파일 원본 데이터

func receiveFile(r io.Reader, saveDir string) error {
	// 4-1. 파일 메타 수신
	metaBytes, err := readWithLength(r)
	if err != nil {
		return fmt.Errorf("read file meta: %w", err)
	}
	var meta dtos.FileInfo
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return fmt.Errorf("unmarshal file meta: %w", err)
	}

	// 4-2. 파일 크기 수신
	sizeBuf := make([]byte, 8)
	if _, err := io.ReadFull(r, sizeBuf); err != nil {
		return fmt.Errorf("read file size: %w", err)
	}
	fileSize := binary.BigEndian.Uint64(sizeBuf)

	// 4-3. 저장 파일 생성 (경로 트래버설 방지: Base만 사용)
	savePath := filepath.Join(saveDir, filepath.Base(meta.Name))
	f, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("create file %s: %w", savePath, err)
	}
	defer f.Close()

	// 4-4. 정확히 fileSize 바이트만 읽어서 저장
	if _, err := io.CopyN(f, r, int64(fileSize)); err != nil {
		return fmt.Errorf("write file data %s: %w", meta.Name, err)
	}

	fmt.Printf("[recv] saved: %s (%d bytes)\n", savePath, fileSize)
	return nil
}

// ── 공통 유틸 ─────────────────────────────────────────────────────────────

// readWithLength : [4B 길이] 를 먼저 읽고 정확히 N바이트를 읽어 반환합니다.
func readWithLength(r io.Reader) ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, fmt.Errorf("read length: %w", err)
	}
	length := binary.BigEndian.Uint32(lenBuf)
	if length == 0 {
		return []byte{}, nil
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("read data (expected %d bytes): %w", length, err)
	}
	return data, nil
}
