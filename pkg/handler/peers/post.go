package peers

import (
	"Client_Agent/pkg/consts"
	"Client_Agent/pkg/dtos"
	"Client_Agent/pkg/session"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// POST /peers/:id
func (h *PeersHandler) Post(c *gin.Context) {
	id := c.Param("id")

	// ── 1. 요청 바디 바인딩 ──────────────────────────────────────────────
	var sendDataRequest dtos.SendDataRequest
	if err := c.ShouldBindJSON(&sendDataRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			consts.KeyECode:   consts.ErrCodeInvalidParam,
			consts.KeyMessage: consts.MsgInvalidRequest,
			consts.KeyDetail:  sendDataRequest,
		})
		return
	}

	// ── 2. 대상 Peer 연결(Conn) 조회 ────────────────────────────────────
	h.service.Sessions.ConnectionsMu.RLock()
	libP2PConn, ok := h.service.Sessions.Connections[id]
	h.service.Sessions.ConnectionsMu.RUnlock()

	if !ok || libP2PConn == nil {
		c.JSON(http.StatusNotFound, gin.H{
			consts.KeyECode:   consts.ErrCodeNotFound,
			consts.KeyMessage: fmt.Sprintf("peer %s not connected", id),
		})
		return
	}

	// ── 3. 전송 ID 생성 ──────────────────────────────────────────────────
	transferId := uuid.New().String()

	// ── 4. Stream 열기 ───────────────────────────────────────────────────
	rawStream, err := libP2PConn.Conn.NewStream(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			consts.KeyECode:   consts.ErrCodeInternal,
			consts.KeyMessage: "failed to open stream",
			consts.KeyDetail:  err.Error(),
		})
		return
	}

	// ── 5. CountingStream으로 래핑 ───────────────────────────────────────
	cs := &session.CountingStream{Stream: rawStream}

	// ── 6. Store에 SendingFile 등록 (전송 시작 전) ───────────────────────
	// 파일이 여러 개여도 하나의 스트림으로 묶어서 관리합니다.
	// key: transferId (peer ID 하나에 여러 전송이 가능하므로)
	sendingFile := session.SendingFile{
		TransferId: transferId,
		PeerID:     id,
		Status:     session.TransferStatusPending,
		TotalFiles: len(sendDataRequest.Files),
		Files:      sendDataRequest.Files,
		Stream:     cs,
	}

	h.service.Sessions.SetSendingFile(transferId, sendingFile)

	// ── 7. 비동기 전송 시작 ──────────────────────────────────────────────
	go func() {
		defer rawStream.Close()

		// 7-1. 상태 → Sending 으로 변경
		h.service.Sessions.UpdateSendingFileStatus(transferId, session.TransferStatusSending)

		// ── 속도 샘플링 고루틴 (전송 고루틴과 생명주기 공유) ────────────────
		stopSampler := make(chan struct{})
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					h.service.Sessions.SendingFilesMu.RLock()
					sf, ok := h.service.Sessions.SendingFiles[transferId]
					h.service.Sessions.SendingFilesMu.RUnlock()
					if ok {
						sf.RecordSpeedSample()
					}
				case <-stopSampler:
					return
				}
			}
		}()
		defer close(stopSampler) // 전송 고루틴 종료 시 샘플러도 자동 종료

		// 7-2. TransferHeader 전송
		type TransferHeader struct {
			TransferId string          `json:"transferId"`
			Files      []dtos.FileInfo `json:"files"`
		}
		headerBytes, err := json.Marshal(TransferHeader{
			TransferId: transferId,
			Files:      sendDataRequest.Files,
		})
		if err != nil {
			h.service.Sessions.UpdateSendingFileStatus(transferId, session.TransferStatusFailed)
			return
		}
		if err := writeWithLength(cs, headerBytes); err != nil {
			h.service.Sessions.UpdateSendingFileStatus(transferId, session.TransferStatusFailed)
			return
		}

		// 7-3. 파일 순차 전송
		for i, fileInfo := range sendDataRequest.Files {
			if err := sendFile(cs, fileInfo); err != nil {
				h.service.Sessions.UpdateSendingFileStatus(transferId, session.TransferStatusFailed)
				return
			}
			// 파일 하나 완료될 때마다 진행 상황 업데이트
			h.service.Sessions.UpdateSendingFileProgress(transferId, i+1)
		}

		// 7-4. 전송 완료
		h.service.Sessions.UpdateSendingFileStatus(transferId, session.TransferStatusDone)
	}()

	// ── 8. 비동기이므로 즉시 202 Accepted 반환 ───────────────────────────
	c.JSON(http.StatusAccepted, dtos.SendDataResponse{
		TransferId: transferId,
		Message:    "transfer started",
	})
}

// sendFile : 단일 파일을 스트림으로 전송합니다.
//
// 전송 프로토콜 (per-file):
//
//	[4B]  fileMetaLen — FileInfo JSON 길이
//	[NB]  fileMeta    — FileInfo JSON
//	[8B]  fileSize    — 실제 파일 바이트 크기 (big-endian uint64)
//	[MB]  fileData    — 파일 원본 데이터
func sendFile(w io.Writer, fileInfo dtos.FileInfo) error {
	// 파일 메타 전송
	metaBytes, err := json.Marshal(fileInfo)
	if err != nil {
		return fmt.Errorf("marshal file meta: %w", err)
	}
	if err := writeWithLength(w, metaBytes); err != nil {
		return fmt.Errorf("write file meta: %w", err)
	}

	// 파일 열기
	f, err := os.Open(fileInfo.Path)
	if err != nil {
		return fmt.Errorf("open file %s: %w", fileInfo.Path, err)
	}
	defer f.Close()

	// 파일 크기 전송 (8 byte big-endian)
	sizeBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeBuf, fileInfo.Size)
	if _, err := w.Write(sizeBuf); err != nil {
		return fmt.Errorf("write file size: %w", err)
	}

	// 파일 데이터 스트리밍
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("stream file data %s: %w", fileInfo.Name, err)
	}

	return nil
}

// writeWithLength : [4B big-endian 길이] + [데이터] 형식으로 기록합니다.
func writeWithLength(w io.Writer, data []byte) error {
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := w.Write(lenBuf); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}
