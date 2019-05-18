package main

import . "airviz/latest"

type Trigger struct {
	Topic
	Index
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Trigger stream to push information to clients, 1 trigger = 1 item added at the given index, of the given topic
	triggers chan Trigger

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		triggers:  make(chan Trigger),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
		case t := <-h.triggers:
			for client := range h.clients {
				client.Trigger(t)
			}
		}
	}
}
