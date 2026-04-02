package transfer

import (
	"Client_Agent/pkg/consts"
	"Client_Agent/pkg/dtos"
	"Client_Agent/pkg/session"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GET /transfer/progress
func (h *TransferStatusHandler) Get(c *gin.Context) {

	// ── 1. 진행 중인 전송 탐색 (Sending 우선, 없으면 Receiving) ──────────
	h.service.Sessions.SendingFilesMu.RLock()
	var activeSending *session.SendingFile
	for _, sf := range h.service.Sessions.SendingFiles {
		if sf.Status == session.TransferStatusSending || sf.Status == session.TransferStatusPending {
			activeSending = sf
			break
		}
	}
	h.service.Sessions.SendingFilesMu.RUnlock()

	h.service.Sessions.ReceivingFilesMu.RLock()
	var activeReceiving *session.ReceivingFile
	for _, rf := range h.service.Sessions.ReceivingFiles {
		if rf.Status == session.TransferStatusSending || rf.Status == session.TransferStatusPending {
			activeReceiving = rf
			break
		}
	}
	h.service.Sessions.ReceivingFilesMu.RUnlock()

	// ── 2. 둘 다 없으면 404 ───────────────────────────────────────────────
	if activeSending == nil && activeReceiving == nil {
		c.JSON(http.StatusNotFound, gin.H{
			consts.KeyECode:   consts.ErrCodeNotFound,
			consts.KeyMessage: "진행 중인 전송이 없습니다.",
		})
		return
	}

	// ── 3. 공통 계산 로직 (방향에 따라 분기) ─────────────────────────────
	type FileProgressItem struct {
		PeerID           string  `json:"peerId"`
		Name             string  `json:"name"`
		Size             int64   `json:"size"`
		TransferredBytes int64   `json:"transferredBytes"`
		Progress         float64 `json:"progress"`
		Speed            int64   `json:"speed"`
	}

	var (
		status       string
		peerID       string
		files        []dtos.FileInfo
		totalBytes   int64 // Stream 전체 전송량
		totalSize    int64
		currentSpeed int64
		avgSpeed     int64
		startedAt    *time.Time
	)

	if activeSending != nil {
		// 송신
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
		// 수신
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

	// ── 4. 파일별 진행 상황 계산 ─────────────────────────────────────────
	// 파일 단위 CountingStream 이 없으므로
	// 전체 전송 바이트를 파일 크기 비율로 안분합니다.
	for _, f := range files {
		totalSize += int64(f.Size)
	}

	var (
		fileItems        []FileProgressItem
		totalTransferred int64
		bytesRemaining   = totalBytes // 파일 순서대로 소진
	)

	for _, f := range files {
		size := int64(f.Size)

		// 이 파일에 할당된 전송 바이트: 남은 전송량과 파일 크기 중 작은 값
		var transferred int64
		if bytesRemaining >= size {
			transferred = size // 이 파일은 완전히 전송됨
			bytesRemaining -= size
		} else {
			transferred = bytesRemaining // 현재 이 파일을 전송 중
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

	// ── 5. 전체 진행률 / ETA ──────────────────────────────────────────────
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

	// ── 6. 응답 반환 ──────────────────────────────────────────────────────
	c.JSON(http.StatusOK, gin.H{
		"status":           status,
		"files":            fileItems,
		"totalSize":        totalSize,
		"totalTransferred": totalTransferred,
		"totalProgress":    totalProgress,
		"currentSpeed":     currentSpeed,
		"averageSpeed":     avgSpeed,
		"eta":              eta,
		"startedAt":        startedAt,
	})
}

// ── 상태 매핑 헬퍼 ────────────────────────────────────────────────────────
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
