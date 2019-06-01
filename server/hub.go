package server

import (
	. "airviz/core"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Trigger stream to push information to clients, 1 trigger = 1 item added at the given index, of the given topic
	Triggers chan Trigger

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client


	idMutex sync.Mutex
	clientIdCounter uint64
}

func NewHub() *Hub {
	return &Hub{
		Triggers:        make(chan Trigger),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		clients:         make(map[*Client]bool),
		clientIdCounter: 1,
	}
}

func (h *Hub) NewClientId() uint64 {
	h.idMutex.Lock()
	id := h.clientIdCounter
	h.clientIdCounter += 1
	h.idMutex.Unlock()
	return id
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
		case t := <-h.Triggers:
			for client := range h.clients {
				client.clientState.Trigger(t)
			}
		}
	}
}
