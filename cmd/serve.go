package cmd

import (
	"context"
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

type Client struct {
	isOnline bool
	conn     *net.TCPConn
}

var clients = make(map[string]Client)
var mu sync.Mutex

func executeServeCommand(_ *cobra.Command, _ []string) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

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
	logrus.WithField("server_addr", serverAddress).Infoln("Запущен обработчик broadcast UDP пакетов")
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
					logrus.WithError(err).Error("Ошибка обработки UDP пакета")
				}
				continue
			}
			logrus.
				WithField("from", clientAddress).
				WithField("body", string(inputBytes)).
				Infoln("Получено UDP сообщение")
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
		logrus.WithError(err).
			WithField("ip", clientAddr.IP.String()).
			Error("Ошибка определения IP пользователя из UDP пакета")
		return
	}
	logrus.WithField("client_addr", rtcpAddr.String()).Println("Определен клиентский адрес")
	tcpConn, err := net.DialTCP("tcp", nil, rtcpAddr)
	if err != nil {
		logrus.WithError(err).WithField("client_addr", rtcpAddr.String()).Errorln("Не удалось подключиться к клиенту")
		return
	}
	mu.Lock()
	_, exist := clients[StartMes[0]]
	if !exist {
		clients[StartMes[0]] = Client{
			isOnline: true,
			conn:     tcpConn,
		}
		logrus.WithFields(logrus.Fields{
			"client_addr": rtcpAddr.String(),
			"username":    StartMes[0],
		}).Infoln("Новый клиент добавлен")
	} else {
		logrus.WithFields(logrus.Fields{
			"client_addr": rtcpAddr.String(),
			"username":    StartMes[0],
		}).Infoln("Клиент с таким username уже существует, добавление пропущено")
	}
	mu.Unlock()
	updateUsersInOnline(tcpConn, StartMes[0])

	defer func() {
		_ = tcpConn.Close()
		// Отправляем всем уведомление о том, что пользователь отключился
		err = sendUpdateDelUser(StartMes[0])
		if err != nil {
			logrus.WithError(err).Errorln("Не удалось отправить уведомление об отключении пользователя")
		}

		mu.Lock()
		defer mu.Unlock()
		delete(clients, StartMes[0])
	}()

	for {
		buf := make([]byte, 1024)
		var n int
		n, err = tcpConn.Read(buf)
		if err != nil {
			break
		}
		logrus.WithFields(logrus.Fields{
			"message":  string(buf[0:n]),
			"username": StartMes[0],
		}).Infoln("Получено сообщение от пользователя")
		newmessage := strings.Split(string(buf), ";;;")
		mu.Lock()
		client, exist := clients[newmessage[0]]
		if exist && client.isOnline {
			_ = sendToUser(client.conn, newmessage[0]+";;;"+newmessage[2])
		}
		mu.Unlock()
	}
}

// updateUsersInOnline обновляет список онлайн пользователей у всех пользователей
func updateUsersInOnline(tcpConn *net.TCPConn, selfUsername string) {
	mu.Lock()
	defer mu.Unlock()
	for index, _ := range clients {
		if index == selfUsername {
			continue
		}
		mes := "server;;;" + "nu;;;" + selfUsername
		_ = sendToUser(tcpConn, mes)
	}
}

// sendUpdateDelUser отправляет всем текущим клиентам уведомление, что клиент отключился
func sendUpdateDelUser(username string) error {
	mu.Lock()
	defer mu.Unlock()
	for index, _ := range clients {
		if index == username {
			continue
		}
		mes := "server;;;" + "du;;;" + username
		err := sendToUser(clients[index].conn, mes)
		if err != nil {
			return err
		}
	}

	return nil
}

func sendToUser(tcpConn *net.TCPConn, message string) error {
	logrus.WithFields(logrus.Fields{
		"message": message,
		"ip":      tcpConn.RemoteAddr().String(),
	}).Println("Send to user")
	_, err := tcpConn.Write([]byte(message))

	return err
}
