package cmd

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
	"os"
	"os/signal"
	"strings"
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

func executeServeCommand(_ *cobra.Command, _ []string) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	clients := NewAtomicClientsMap()

	serverAddress, err := net.ResolveUDPAddr("udp4", "255.255.255.255:8889")
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
			err = connection.SetReadDeadline(time.Now().Add(1 * time.Second))
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
				}).Println("set read deadline error")
				return
			}

			var message UdpMessage
			err = json.NewDecoder(connection).Decode(&message)
			if err != nil {
				if !strings.Contains(err.Error(), "i/o timeout") {
					logrus.WithFields(logrus.Fields{
						"error": err,
					}).Errorln("read decode error")
				}
				continue
			}

			logrus.WithFields(logrus.Fields{
				"username": message.Username,
				"ip":       message.ClientIp,
				"port":     message.ClientPort,
			}).Println("New Client")
			client, err := InitClient(
				message.Username,
				message.ClientIp,
				message.ClientPort,
			)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
				}).Errorln("init client error")
			}
			logrus.WithFields(logrus.Fields{
				"client": client,
			}).Println("New Client!!!")
			go client.handleTunnelClient(ctx, clients)
		}
	}
}

//func updateUsersInOnline(tcpConn *net.TCPConn, selfUsername string) {
//	mu.Lock()
//	for index, _ := range clients {
//		logrus.WithFields(logrus.Fields{
//			"index": index,
//		}).Println("")
//		if index == selfUsername {
//			continue
//		}
//		mes := "server;;;" + "nu;;;" + selfUsername
//		sendToUser(tcpConn, mes)
//	}
//	mu.Unlock()
//}
//
//func sendToUser(tcpConn *net.TCPConn, message string) {
//	logrus.WithFields(logrus.Fields{
//		"message": message,
//		"ip":      tcpConn.RemoteAddr().String(),
//	}).Println("Send to user")
//	_, err := tcpConn.Write([]byte(message))
//	if err != nil {
//		logrus.Error(err)
//	}
//}
