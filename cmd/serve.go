package cmd

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "WebSocket мессенджер",
	Long:  `Запускает веб-сервер для прослушки определенного порта и обработки вебхуков для клиентов`,
	Run:   executeServeCommand,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

type Message struct {
	TargetIP   string `json:"target_ip"`
	TargetPort int    `json:"target_port"`
	Data       string `json:"data"`
}

type Client struct {
	isOnline bool
	conn     *net.TCPConn
}

var clients = make(map[string]Client)
var mu sync.Mutex

func executeServeCommand(_ *cobra.Command, _ []string) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	serverAddress, err := net.ResolveUDPAddr("udp4", "192.168.0.255:8889")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err,
		}).Println("resolve udp address error")
		return
	}
	connection, err := net.ListenUDP("udp", serverAddress)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Println("start listen error")
		return
	}
	defer connection.Close()
	for {
		select {
		case <-ctx.Done():
			stop()
			return
		default:
			err := connection.SetReadDeadline(time.Now().Add(1 * time.Second))
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
				}).Println("set read deadline error")
				return
			}

			inputBytes := make([]byte, 1024)
			n, clientAddress, err := connection.ReadFromUDP(inputBytes)
			if err != nil {
				if !strings.Contains(err.Error(), "i/o timeout") {
					logrus.WithFields(logrus.Fields{
						"error": err,
					}).Println("read error")
				}
				continue
			}
			logrus.WithFields(logrus.Fields{
				"inputBytes": string(inputBytes),
				"clientAddr": clientAddress,
				"n":          n,
			}).Println("Received message")
			go handleTunnelClient(clientAddress, inputBytes[:n])
		}
	}
}

func handleTunnelClient(clientAddr *net.UDPAddr, data []byte) {
	StartMes := strings.Split(string(data), ";;;")
	logrus.WithFields(logrus.Fields{
		"StartMes": StartMes,
		"con":      clientAddr.IP.String() + ":" + StartMes[1],
	}).Println("Connecting to client")
	rtcpAddr, err := net.ResolveTCPAddr("tcp", clientAddr.IP.String()+":8888")
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Println(rtcpAddr)
	tcpConn, err := net.DialTCP("tcp", nil, rtcpAddr)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Println(tcpConn)
	mu.Lock()
	_, exist := clients[StartMes[0]]
	logrus.Println("exist: ", exist)
	if !exist {
		clients[StartMes[0]] = Client{
			isOnline: true,
			conn:     tcpConn,
		}
		logrus.WithFields(logrus.Fields{
			"ip":       clientAddr.IP.String(),
			"username": StartMes[0],
		}).Println("Create new client")
	}
	mu.Unlock()
	updateUsersInOnline(tcpConn, StartMes[0])
	defer tcpConn.Close()
	for {
		buf := make([]byte, 1024)
		var n int
		n, err = tcpConn.Read(buf)
		if err != nil {
			break
		}
		fmt.Print("Message Received:", string(buf[0:n]), "\n")
		newmessage := strings.Split(string(buf), ";;;")
		fmt.Println(newmessage)
		mu.Lock()
		client, exist := clients[newmessage[0]]
		if exist && client.isOnline {
			sendToUser(client.conn, newmessage[0]+";;;"+newmessage[2])
		}
		mu.Unlock()
	}
}

func updateUsersInOnline(tcpConn *net.TCPConn, selfUsername string) {
	mu.Lock()
	for index, _ := range clients {
		logrus.WithFields(logrus.Fields{
			"index": index,
		}).Println("")
		if index == selfUsername {
			continue
		}
		mes := "server;;;" + "nu;;;" + selfUsername
		sendToUser(tcpConn, mes)
	}
	mu.Unlock()
}

func sendToUser(tcpConn *net.TCPConn, message string) {
	logrus.WithFields(logrus.Fields{
		"message": message,
		"ip":      tcpConn.RemoteAddr().String(),
	}).Println("Send to user")
	_, err := tcpConn.Write([]byte(message))
	if err != nil {
		logrus.Error(err)
	}
}
