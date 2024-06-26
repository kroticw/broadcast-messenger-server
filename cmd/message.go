package cmd

type TcpMessage struct {
	From        string `json:"from"`
	To          string `json:"to"`
	ServiceType string `json:"servtype"`
	ServiceData string `json:"servdata"`
	Message     string `json:"message"`
}

type UdpMessage struct {
	ClientIp   string `json:"client_ip"`
	ClientPort string `json:"client_port"`
	Username   string `json:"username"`
}
