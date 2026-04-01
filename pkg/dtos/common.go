package dtos

type Peer struct {
	Nickname       string `json:"nickname"`
	Address        string `json:"address"`
	ConnectionType string `json:"connectionType"`
	RTT            int64  `json:"rtt"`
}
