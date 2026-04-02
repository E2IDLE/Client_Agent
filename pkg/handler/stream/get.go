package stream

import (
	"Client_Agent/pkg/consts"
	"Client_Agent/pkg/dtos"
	"Client_Agent/pkg/session"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GET /stream
func (h *StreamHandler) Get(c *gin.Context) {

	// ── 1. SSE 헤더 설정 ─────────────────────────────────────────────────
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*") // 필요 시 도메인 제한
	c.Status(http.StatusOK)

	// ── 2. 클라이언트 연결 해제 감지 ─────────────────────────────────────
	clientGone := c.Request.Context().Done()

	// ── 3. flush 가능한 writer 확보 ──────────────────────────────────────
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			consts.KeyECode:   consts.ErrCodeInternal,
			consts.KeyMessage: "streaming not supported",
		})
		return
	}

	// ── 4. SSE 이벤트 전송 헬퍼 ──────────────────────────────────────────
	sendEvent := func(event string, data any) error {
		payload, err := json.Marshal(data)
		if err != nil {
			return err
		}
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, string(payload))
		flusher.Flush()
		return nil
	}

	// ── 5. 연결 확인용 초기 이벤트 ───────────────────────────────────────
	sendEvent("connected", gin.H{"message": "stream connected"})

	// ── 6. 1초마다 진행 상황 전송 ────────────────────────────────────────
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-clientGone:
			// 클라이언트 연결 해제
			return

		case <-ticker.C:
			payload := h.buildProgressPayload()
			if payload == nil {
				// 진행 중인 전송 없음 → idle 이벤트
				sendEvent("idle", gin.H{"message": "no active transfer"})
				continue
			}
			if err := sendEvent("progress", payload); err != nil {
				return
			}

			// 완료/실패/취소 상태면 done 이벤트 후 연결 종료
			status, _ := payload["status"].(string)
			if status == "completed" || status == "failed" || status == "cancelled" {
				sendEvent("done", gin.H{"status": status})
				return
			}
		}
	}
}

// ── 진행 상황 페이로드 빌드 ───────────────────────────────────────────────
func (h *StreamHandler) buildProgressPayload() gin.H {

	// Sending 우선 탐색
	h.service.Sessions.SendingFilesMu.RLock()
	var activeSending *session.SendingFile
	for _, sf := range h.service.Sessions.SendingFiles {
		if sf.Status == session.TransferStatusPending ||
			sf.Status == session.TransferStatusSending {
			activeSending = sf
			break
		}
	}
	h.service.Sessions.SendingFilesMu.RUnlock()

	// Receiving 탐색
	h.service.Sessions.ReceivingFilesMu.RLock()
	var activeReceiving *session.ReceivingFile
	for _, rf := range h.service.Sessions.ReceivingFiles {
		if rf.Status == session.TransferStatusPending ||
			rf.Status == session.TransferStatusSending {
			activeReceiving = rf
			break
		}
	}
	h.service.Sessions.ReceivingFilesMu.RUnlock()

	if activeSending == nil && activeReceiving == nil {
		return nil
	}

	// ── 공통 변수 추출 ────────────────────────────────────────────────────
	type FileProgressItem struct {
		PeerID           string  `json:"peerId"`
		Name             string  `json:"name"`
		Size             int64   `json:"size"`
		TransferredBytes int64   `json:"transferredBytes"`
		Progress         float64 `json:"progress"`
		Speed            int64   `json:"speed"`
	}

	var (
		direction    string
		transferId   string
		status       string
		peerID       string
		files        []dtos.FileInfo
		totalBytes   int64
		currentSpeed int64
		avgSpeed     int64
		startedAt    *time.Time
	)

	if activeSending != nil {
		direction = "sending"
		transferId = activeSending.TransferId
		status = toAPIStatus(activeSending.Status)
		peerID = activeSending.PeerID
		files = activeSending.Files
		totalBytes = activeSending.Stream.BytesSent()
		currentSpeed = activeSending.CurrentSpeed()
		avgSpeed = activeSending.AverageSpeed()
		if !activeSending.StartedAt.IsZero() {
			startedAt = &activeSending.StartedAt
		}
	} else {
		direction = "receiving"
		transferId = activeReceiving.TransferId
		status = toAPIStatus(activeReceiving.Status)
		peerID = activeReceiving.PeerID
		files = activeReceiving.Files
		totalBytes = activeReceiving.Stream.BytesReceived()
		currentSpeed = activeReceiving.CurrentSpeed()
		avgSpeed = activeReceiving.AverageSpeed()
		if !activeReceiving.StartedAt.IsZero() {
			startedAt = &activeReceiving.StartedAt
		}
	}

	// ── 파일별 진행 상황 계산 ─────────────────────────────────────────────
	var totalSize int64
	for _, f := range files {
		totalSize += int64(f.Size)
	}

	var (
		fileItems        []FileProgressItem
		totalTransferred int64
		bytesRemaining   = totalBytes
	)

	for _, f := range files {
		size := int64(f.Size)

		var transferred int64
		if bytesRemaining >= size {
			transferred = size
			bytesRemaining -= size
		} else {
			transferred = bytesRemaining
			bytesRemaining = 0
		}
		totalTransferred += transferred

		var progress float64
		if size > 0 {
			progress = float64(transferred) / float64(size) * 100
			if progress > 100 {
				progress = 100
			}
		}

		var fileSpeed int64
		if totalSize > 0 {
			fileSpeed = int64(float64(currentSpeed) * float64(size) / float64(totalSize))
		}

		fileItems = append(fileItems, FileProgressItem{
			PeerID:           peerID,
			Name:             f.Name,
			Size:             size,
			TransferredBytes: transferred,
			Progress:         progress,
			Speed:            fileSpeed,
		})
	}

	// ── 전체 진행률 / ETA ─────────────────────────────────────────────────
	var totalProgress float64
	if totalSize > 0 {
		totalProgress = float64(totalTransferred) / float64(totalSize) * 100
		if totalProgress > 100 {
			totalProgress = 100
		}
	}

	var eta *int64
	remaining := totalSize - totalTransferred
	if avgSpeed > 0 && remaining > 0 {
		etaVal := remaining / avgSpeed
		eta = &etaVal
	}

	return gin.H{
		"transferId":       transferId,
		"direction":        direction,
		"status":           status,
		"files":            fileItems,
		"totalSize":        totalSize,
		"totalTransferred": totalTransferred,
		"totalProgress":    totalProgress,
		"currentSpeed":     currentSpeed,
		"averageSpeed":     avgSpeed,
		"eta":              eta,
		"startedAt":        startedAt,
	}
}

func toAPIStatus(s session.TransferStatus) string {
	switch s {
	case session.TransferStatusPending:
		return "preparing"
	case session.TransferStatusSending:
		return "transferring"
	case session.TransferStatusDone:
		return "completed"
	case session.TransferStatusFailed:
		return "failed"
	case session.TransferStatusCancelled:
		return "cancelled"
	default:
		return "preparing"
	}
}
