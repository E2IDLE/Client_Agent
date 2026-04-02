package dtos

type Peer struct {
	Id             string
	Nickname       string `json:"nickname"`
	Address        string `json:"address"`
	ConnectionType string `json:"connectionType"`
	RTT            uint64 `json:"rtt"`
}
