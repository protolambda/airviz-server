package server

import (
	. "airviz/core"
	"encoding/binary"
	"fmt"
)

type ClientState struct {
	blocks *DataRequestHandler
}

func (cs *ClientState) HandleEvents() {
	go cs.blocks.handleRequests()
	go cs.blocks.pushesToRequests()
}

func (cs *ClientState) OnMessage(message []byte) {
	if len(message) <= 1 + 4 {
		return
	}
	topic := binary.LittleEndian.Uint32(message[0:4])
	start := Index(binary.LittleEndian.Uint32(message[4:8]))
	windowLen := (len(message) - 8) / 4
	data := make([]uint32, windowLen, windowLen)
	j := 0
	for i := 8; i < len(message); i += 4 {
		data[j] = binary.LittleEndian.Uint32(message[i:i+4])
		j++
	}

	switch Topic(topic) {
	case TopicBlocks:
		cs.blocks.updateStatus(start, data)
		cs.blocks.makeRequest(start, start + Index(len(data)))
	default:
		fmt.Printf("Warning: unhandled topic: %d\n", topic)
	}
}

func (cs *ClientState) Trigger(t Trigger) {
	switch t.Topic {
	case TopicBlocks:
		cs.blocks.pushes <- t.Index
	default:
		fmt.Printf("Warning: unhandled trigger: topic: %d index: %d\n", t.Topic, t.Index)
	}
}

func (cs *ClientState) Close() {
	cs.blocks.Close()
}

