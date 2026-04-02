package dtos

type AgentStatusResponse struct {
	AgentName     string `json:"agentName"`
	AgentVersion  string `json:"agentVersion"`
	Status        string `json:"status"`
	Uptime        string `json:"uptime"`
	PeerID        string `json:"peerId"`
	MultiAddress  string `json:"multiAddress"`
	NATType       string `json:"natType"`
	ConnectedPeer []Peer `json:"connectedPeer"`
}

type PatchAgentStatusRequest struct {
	Name string `json:"name"`
}
