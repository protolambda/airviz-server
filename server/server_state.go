package server

import (
	. "airviz/core"
	"airviz/datasrc"
	. "airviz/latest"
	"fmt"
	"log"
	"net/http"
)

type ServerDags map[Topic]*Dag

type ServerState struct {
	hub *Hub
	dags ServerDags
	events chan datasrc.DataEvent
}

func NewServerState(hub *Hub) *ServerState {
	return &ServerState{
		hub: hub,
		dags: make(ServerDags),
		events: make(chan datasrc.DataEvent, 100),
	}
}

func (s *ServerState) GetEventCh() chan<- datasrc.DataEvent {
	return s.events
}

func (s *ServerState) SetDag(topic Topic, dag *Dag) {
	s.dags[topic] = dag
}

func (s *ServerState) newDataRequestHandler(sendMsg func([]byte), topic Topic) *DataRequestHandler {
	if dag, ok := s.dags[topic]; ok {
		return NewDataRequestHandler(sendMsg, dag, topic)
	} else {
		fmt.Printf("cannot create handler, unrecognized topic %d\n", topic)
		return nil
	}
}

// Start serving a new client
func (s *ServerState) ServeWs(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// make a client
	client := &Client{
		id: s.hub.NewClientId(),
		hub: s.hub,
		conn: conn,
		send: make(chan []byte, buffedMsgCount),
	}
	// give it a state
	client.clientState = &ClientState{
		blocks: s.newDataRequestHandler(client.sendMsg, TopicBlocks),
	}

	// register it
	client.hub.register <- client

	// start processing routines for the client
	go client.writePump()
	go client.readPump()
	go client.clientState.HandleEvents()
}

func (s *ServerState) PipeEvents() {
	for {
		ev := <-s.events
		if dag, ok := s.dags[ev.Topic]; ok {
			dag.AddBox(ev.Box)
		} else {
			fmt.Printf("cannot pipe event into dag, unrecognized topic %d\n", ev.Topic)
			continue
		}
		s.hub.Triggers <- Trigger{Topic: ev.Topic, Index: ev.Box.Index}
	}
}
