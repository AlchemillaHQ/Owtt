package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	endpoint := "wss://localhost:8000"

	d := websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	c, _, err := d.Dial(endpoint, nil)
	if err != nil {
		log.Fatalf("Error connecting to WSS endpoint: %v", err)
	}
	defer c.Close()

	payload := `{"action": "scrape"}`

	err = c.WriteMessage(websocket.TextMessage, []byte(payload))
	if err != nil {
		log.Fatalf("Error writing message: %v", err)
	}

	c.SetReadDeadline(time.Now().Add(5 * time.Second))

	_, message, err := c.ReadMessage()
	if err != nil {
		log.Fatalf("Error reading message: %v", err)
	}

	fmt.Printf("Received message: %s\n", message)
}
