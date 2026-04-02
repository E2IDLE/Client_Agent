package transfer

import (
	"Client_Agent/pkg/service"

	"github.com/gin-gonic/gin"
)

const (
	PathTransferProgress = "/transfer/progress"
	PathTransferCancel   = "/transfer/cancel"
)

type TransferStatusHandler struct {
	service *service.TransferStatusService
}

func NewTransferStatusHandler(svc *service.TransferStatusService) *TransferStatusHandler {
	return &TransferStatusHandler{service: svc}
}

func (h *TransferStatusHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET(PathTransferProgress, h.Get)
	rg.PATCH(PathTransferProgress, h.Patch)
	rg.POST(PathTransferCancel, h.Cancel)
}
