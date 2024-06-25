package cmd

import (
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type Client struct {
	ClientIP   string `json:"client_ip"`
	ClientPort string `json:"client_port"`
	Username   string `json:"username"`
	Conn       *net.TCPConn
}

type AtomicClientsMap struct {
	mu         sync.Mutex
	clientsMap map[string]*Client
}

func NewAtomicClientsMap() *AtomicClientsMap {
	return &AtomicClientsMap{
		clientsMap: make(map[string]*Client),
	}
}

func (a *AtomicClientsMap) Get(username string) (*Client, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	client, ok := a.clientsMap[username]
	return client, ok
}

func (a *AtomicClientsMap) Push(client *Client) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.clientsMap[client.Username] = client
}

func (a *AtomicClientsMap) Remove(username string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.clientsMap, username)
}

func (a *AtomicClientsMap) GetMap() map[string]*Client {
	a.mu.Lock()
	defer a.mu.Unlock()
	cm := a.clientsMap
	return cm
}

func (a *AtomicClientsMap) Len() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.clientsMap)
}

func InitClient(
	username string,
	ip string,
	port string,
) (*Client, error) {
	logrus.WithFields(logrus.Fields{
		"username": username,
	}).Println("Starting tcp client")
	rtcpAddr, err := net.ResolveTCPAddr("tcp", ip+":"+port)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"username":  username,
			"client_ip": ip,
			"err":       err,
		}).Errorln("Error defer c.Conn.Close()resolving tcp address")
		return &Client{}, err
	}
	logrus.WithFields(logrus.Fields{
		"username": username,
	}).Println("Connecting to client...")
	tcpConn, err := net.DialTCP("tcp", nil, rtcpAddr)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"username":  username,
			"client_ip": ip,
			"err":       err,
		}).Errorln("Error connecting to client")
		return &Client{}, err
	}
	logrus.WithFields(logrus.Fields{
		"username": username,
	}).Println("Connecting to client successfully...")

	return &Client{
		ClientIP:   ip,
		ClientPort: port,
		Username:   username,
		Conn:       tcpConn,
	}, nil
}

func (c *Client) Close(clients *AtomicClientsMap) {
	defer c.Conn.Close()
	clients.Remove(c.Username)
	logrus.Infof("client %s closed successfully", c.Username)
}
