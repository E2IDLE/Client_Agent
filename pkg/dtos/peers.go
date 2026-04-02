package dtos

type PostPeersRequest struct {
	PeerId           string `json:"peerId"`
	PeerMultiAddress string `json:"peerMultiAddress"`
}

type PeerInfo struct {
	PeerId         string `json:"peerId"`
	Nickname       string `json:"nickname"`
	ConnectionType string `json:"connectionType"`
	Status         string `json:"status"`
	Rtt            uint64 `json:"rtt"`
	PacketLoss     string `json:"packetLoss"`
	MultiAddress   string `json:"multiAddress"`
}

type SendDataResponse struct {
	TransferId string `json:"transferId"`
	Message    string `json:"message"`
}

type FileInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     uint64 `json:"size"`
	MimeType string `json:"mimeType"`
}

type SendDataRequest struct {
	Files []FileInfo `json:"files"`
}
