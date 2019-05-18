package main

import (
	. "airviz/latest"
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	// Maximum amounts of messages to buffer to a client before disconnecting them
	buffedMsgCount = 20

)

type Topic byte

const (
	topicDefault Topic = iota
	topicBlocks
	topicState
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var newline = []byte{'\n'}

// Client is a middleman between the websocket connection and the hub.
type Client struct {

	id uint64

	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	blocks *DataRequestHandler

}

func (c *Client) Trigger(t Trigger) {
	switch t.Topic {
	case topicBlocks:
		c.blocks.pushes <- t.Index
	default:
		fmt.Printf("Warning: unhandled trigger: topic: %d index: %d\n", t.Topic, t.Index)
	}
}

func (c *Client) Close() {
	c.blocks.Close()
	close(c.send)
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		if err := c.conn.Close(); err != nil {
			fmt.Printf("Client %d unregistered with an error: %v\n", c.id, err)
		}
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		if len(message) <= 1 + 4 {
			continue
		}
		topic := message[0]
		start := Index(binary.BigEndian.Uint32(message[1:5]))
		windowLen := (len(message) - 5) / 4
		data := make([]uint32, windowLen, windowLen)
		j := 0
		for i := 5; i < len(message); i += 4 {
			data[j] = binary.BigEndian.Uint32(message[i:i+4])
			j++
		}

		switch Topic(topic) {
		case topicBlocks:
			c.blocks.updateStatus(start, data)
		default:
			fmt.Printf("Warning: unhandled topic: %d\n", topic)
		}
		c.blocks.makeRequest(start, start + Index(len(data)))
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if err := c.conn.Close(); err != nil {
			fmt.Printf("Stopped connection with client %d, but with an error: %v\n", c.id, err)
		}
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, buffedMsgCount)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
	go client.blocks.handleRequests()
	go client.blocks.pushesToRequests()
}