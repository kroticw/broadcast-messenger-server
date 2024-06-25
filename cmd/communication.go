package cmd

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

var itsFile = false

func (c *Client) handleTunnelClient(ctx context.Context, clients *AtomicClientsMap) {
	defer c.Close(clients)
	_, exist := clients.Get(c.Username)
	logrus.Println("exist: ", exist)
	quit := make(chan bool)
	if !exist {
		clients.Push(c)
		logrus.WithFields(logrus.Fields{
			"ip":       c.ClientIP,
			"username": c.Username,
		}).Println("Create new client")
		var newClientMes TcpMessage
		mp := clients.GetMap()
		for _, client := range mp {
			if client.Username != c.Username {
				newClientMes = TcpMessage{
					From:        "server",
					To:          client.Username,
					ServiceType: "new_user",
					ServiceData: c.Username,
				}
				err := c.sendTo(&newClientMes, clients)
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"ip":       c.ClientIP,
						"username": c.Username,
						"service":  "new_user",
					}).Error(err)
					return
				}
			}
		}

		go c.meInOnline(clients, quit)
	}
	for {
		var message TcpMessage
		err := json.NewDecoder(c.Conn).Decode(&message)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error":    err,
				"ip":       c.ClientIP,
				"username": c.Username,
			}).Errorln("error receive and decode message")
			c.Close(clients)
			return
		}
		logrus.WithFields(logrus.Fields{
			"message":  message,
			"ip":       c.ClientIP,
			"username": c.Username,
		}).Println("Receive message")

		if message.ServiceType == "file" && message.ServiceData != "0" {
			err = c.situationFile(&message, clients)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
				}).Errorln("situation file error")
				c.Close(clients)
				return
			}
		} else {
			err = c.sendTo(&message, clients)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
				})
				c.Close(clients)
				return
			}
		}

	}
}

func (c *Client) situationFile(message *TcpMessage, clients *AtomicClientsMap) error {
	file, err := os.Create(strings.TrimSpace(message.Message))
	if err != nil {
		return err
	}
	length, err := strconv.Atoi(message.ServiceData)
	if err != nil {
		return err
	}
	n, err := io.CopyN(file, c.Conn, int64(length))
	if err != nil {
		return err
	}
	fmt.Println(n, " bytes sent")
	file.Close()
	newMes := &TcpMessage{
		From:        message.From,
		To:          message.To,
		ServiceType: message.ServiceType,
		ServiceData: message.ServiceData,
		Message:     message.Message,
	}
	err = c.sendTo(newMes, clients)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Errorln("error sending message")
		return err
	}
	time.Sleep(100 * time.Millisecond)

	c.sendFileTo(message, clients)
	return nil
}

// функция, которая раз в 5 секунд отправляет всем в clients 2 сообщения - размер json и json, что она новый юзер
func (c *Client) meInOnline(clients *AtomicClientsMap, quit chan bool) {
	//defer c.Close(clients)
	for {
		select {
		case <-quit:
			return
		default:
			mp := clients.GetMap()
			for _, client := range mp {
				if client.Username != c.Username {
					var newClientMes TcpMessage
					if client.Username != c.Username {
						newClientMes = TcpMessage{
							From:        "server",
							To:          client.Username,
							ServiceType: "new_user",
							ServiceData: c.Username,
						}

						err := c.sendTo(&newClientMes, clients)
						if err != nil {
							logrus.WithFields(logrus.Fields{
								"error": err,
							}).Errorln("error sending message")
							return
						}
					}
				}
			}
			time.Sleep(10 * time.Second)
		}
	}
}

func (c *Client) sendTo(message *TcpMessage, clients *AtomicClientsMap) error {
	client, exist := clients.Get(message.To)
	if !exist {
		logrus.Errorln("client not found for sending file")
		return nil
	}
	Json, err := json.Marshal(message)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
			"json":  Json,
		}).Errorln("error marshaling sending message")
	}
	sizeMes := len(Json)
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, uint32(sizeMes))
	_, err = client.Conn.Write(bs)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Errorln("error sending message size")
	}
	time.Sleep(200 * time.Millisecond)
	err = json.NewEncoder(client.Conn).Encode(message)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Errorln("error sending message")
	}
	return nil
}

func (c *Client) sendFileTo(message *TcpMessage, clients *AtomicClientsMap) error {
	logrus.WithFields(logrus.Fields{
		"to": message.To,
	}).Println("start sending file")
	client, exist := clients.Get(message.To)
	if !exist {
		logrus.Errorln("client not found for sending file")
		return nil
	}
	file, err := os.Open(strings.TrimSpace(message.Message))
	if err != nil {
		return err
	}

	length, err := strconv.Atoi(message.ServiceData)
	if err != nil {
		return err
	}

	defer file.Close()
	n, err := io.CopyN(file, client.Conn, int64(length))
	if err != nil {
		return err
	}
	logrus.WithFields(logrus.Fields{
		"message": message,
		"to":      message.To,
		"size":    n,
	}).Println("send file to client")
	return nil
}
