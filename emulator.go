package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	log "gopkg.in/inconshreveable/log15.v2"
)

type chatMessage struct {
	Text      string `json:"text"`
	FirstName string `json:"firstName"`
	UserName  string `json:"userName"`
	UserID    int    `json:"userId"`
	Phone     string `json:"phone"`
	Webhook   string `json:"webhook"`
}

func readBody(req *http.Request, payload interface{}) error {
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(payload)
	if err != nil {
		log.Error("failed to parse body", "err", err)
	}
	return err
}

func forwardMessages(rw http.ResponseWriter, req *http.Request) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	conn, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.Error("failed to upgrade to websockets", "err", err)
		return
	}
	isOpen := true

	go func() {
		defer conn.Close()

		for {
			botResponse := <-messages

			if !isOpen {
				messages <- botResponse
				break
			}

			if err := conn.WriteJSON(&botResponse); err != nil {
				log.Error("failed to write data", "err", err)
				break
			}
		}
	}()

	go func() {
		defer func() {
			isOpen = false
			conn.Close()
		}()

		for {
			var message chatMessage
			if err := conn.ReadJSON(&message); err != nil {
				log.Error("failed to read data", "err", err)
				break
			}
			sendUpdateToBot(message)
		}
	}()
}

func sendUpdateToBot(message chatMessage) {
	update := Update{
		Message: Message{
			Text: message.Text,
			From: User{
				FirstName: message.FirstName,
				ID:        message.UserID,
			},
		},
		UpdateID: 0,
	}

	if _, err := sendJSON(message.Webhook, &update); err != nil {
		log.Error("failed to send update", "err", err)
	}
}

func mockTelegram(rw http.ResponseWriter, req *http.Request) {
	var botResponse botResponse
	readBody(req, &botResponse)

	messages <- botResponse

	response := telegramResponse{
		OK: true,
	}
	json.NewEncoder(rw).Encode(&response)
}
