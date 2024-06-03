package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
	"os"
	"os/signal"
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
	TargetPort string `json:"target_port"`
	Data       string `json:"data"`
}

var clients = make(map[string]*net.TCPConn)
var mu sync.Mutex

func executeServeCommand(_ *cobra.Command, _ []string) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	udpAddr, err := net.ResolveUDPAddr("udp", ":8000")
	if err != nil {
		logrus.Fatal(err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		logrus.Fatal(err)
	}
	defer udpConn.Close()

	buffer := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			stop()
			return
		default:
			err := udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))
			if err != nil {
				logrus.Fatal(err)
				return
			}
			n, clientAddr, err := udpConn.ReadFromUDP(buffer)
			if err != nil {
				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					continue
				}
				logrus.Error(err)
				continue
			}

			go handleClient(clientAddr, buffer[:n])
		}
	}
}

func handleClient(clientAddr *net.UDPAddr, data []byte) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		logrus.Error(err)
		return
	}

	mu.Lock()
	tcpConn, exists := clients[clientAddr.IP.String()]
	if !exists {
		tcpAddr, err := net.ResolveTCPAddr("tcp", clientAddr.IP.String()+":9000")
		if err != nil {
			logrus.Error(err)
			mu.Unlock()
			return
		}

		tcpConn, err = net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			logrus.Error(err)
			mu.Unlock()
			return
		}
		clients[clientAddr.IP.String()] = tcpConn
	}
	mu.Unlock()

	targetAddr := msg.TargetIP + ":" + msg.TargetPort
	mu.Lock()
	targetConn, exists := clients[targetAddr]
	if !exists {
		tcpAddr, err := net.ResolveTCPAddr("tcp", targetAddr)
		if err != nil {
			logrus.Error(err)
			mu.Unlock()
			return
		}

		targetConn, err = net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			logrus.Error(err)
			mu.Unlock()
			return
		}
		clients[targetAddr] = targetConn
	}
	mu.Unlock()

	_, err = targetConn.Write([]byte(msg.Data))
	if err != nil {
		logrus.Error(err)
		return
	}

	logrus.Infof("Сообщение от %s переслано к %s", clientAddr.IP.String(), targetAddr)
}
