package dtos

type PostPeersRequest struct {
	PeerId           string `json:"peerId"`
	PeerMultiAddress string `json:"peerMultiAddress"`
}
