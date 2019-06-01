package server

import (
	. "airviz/core"
	. "airviz/latest"
	"encoding/binary"
	"fmt"
	"sync"
	"time"
)

const defaultWindowSpan Index = 100

type DataRequest struct {
	Start Index
	End   Index
}

type DataRequestHandler struct {
	dag *Dag

	topic Topic

	stat *Status

	pushes chan Index

	lastRequest *DataRequest
	gotRequest  chan bool
	statusUpdateLock sync.Mutex

	send func([]byte)
}

func NewDataRequestHandler(send func([]byte), dag *Dag, topic Topic) *DataRequestHandler {
	return &DataRequestHandler{
		dag:         dag,
		topic:       topic,
		stat:        &Status{Time: 0, Counts: make([]uint32, dag.Length())},
		pushes:      make(chan Index, 256),
		gotRequest:  make(chan bool, 1),
		lastRequest: nil,
		send:        send,
	}
}

func (th *DataRequestHandler) Close() {
	close(th.pushes)
}

func (th *DataRequestHandler) updateStatus(start Index, counts []uint32) {
	// update local status
	th.statusUpdateLock.Lock()
	th.stat.UpdateStatus(start, counts)
	th.statusUpdateLock.Unlock()
}

func (th *DataRequestHandler) makeRequest(start Index, end Index) {
	th.lastRequest = &DataRequest{Start: start, End: end}
	if len(th.gotRequest) == 0 {
		th.gotRequest <- true
	}
}

func (th *DataRequestHandler) makeUpdateMsg(depth uint32, node *DagNode) []byte {
	serialized := node.Box.Value.Serialize()
	// topic, index, depth, padding, parent-root, self-root, serialized value
	msg := make([]byte, 4+4+4+20+32+32+len(serialized))
	binary.LittleEndian.PutUint32(msg[0:8], uint32(th.topic))
	binary.LittleEndian.PutUint32(msg[4:8], uint32(node.Box.Index))
	binary.LittleEndian.PutUint32(msg[8:12], depth)
	copy(msg[32:64], node.Box.ParentKey[:])
	copy(msg[64:96], node.Box.Key[:])
	copy(msg[96:], serialized)
	return msg
}

func (th *DataRequestHandler) handleRequests() {
	for {
		// wait for a trigger before continuing
		<-th.gotRequest
		if th.lastRequest != nil {
			th.statusUpdateLock.Lock()
			updates, err := th.dag.GetStatusUpdate(th.stat, th.lastRequest.Start, th.lastRequest.End)
			if err != nil {
				fmt.Printf("warning: %v\n", err)
			}
			th.statusUpdateLock.Unlock()
			for _, u := range updates {
				th.send(th.makeUpdateMsg(u.Depth, u.Node))
			}
		}
		// wait for a bit before handling new requests.
		time.Sleep(time.Millisecond * 100)
	}
}

// maps pushes into a request, if pushes are relevant (based on last request).
// Buffers pushes together in the span of 1 second.
// Waits for new push (if channel is empty) before continuing converting.
func (th *DataRequestHandler) pushesToRequests() {
	stop := false
	preReadItem := ^Index(0)
	// convert pushes into requests
	for {
		// buffer a bunch of pushes
		todoLen := len(th.pushes)
		hitWindow := false
		checkWindowHit := func(t Index) {
			if th.lastRequest == nil {
				th.lastRequest = &DataRequest{Start: 0, End: defaultWindowSpan}
			}
			if t >= th.lastRequest.Start && t < th.lastRequest.End {
				hitWindow = true
			}
		}
		// we pre-read an item to wait for events, but we don't want to forget about this item
		if preReadItem != ^Index(0) {
			checkWindowHit(preReadItem)
		}
		for i := 0; i < todoLen; i++ {
			t, ok := <-th.pushes
			if !ok {
				// we just closed, stop processing after this
				stop = true
			}
			checkWindowHit(t)
		}
		// if there is no work to do, wait for a bit, and check again
		if !hitWindow {
			// maybe we just need to stop because we can't receive pushes anymore
			if stop {
				return
			}
			// no stopping yet, but no work to do either, wait for an update, then wait another second (to batch pushes), then form a request
			item, ok := <-th.pushes
			// it may also be the last item
			if !ok {
				stop = true
			}
			preReadItem = item
			time.Sleep(time.Second)
			continue
		} else {
			// there are pushes within the client range, repeat the client request.
			// trigger, if it's not already.
			if len(th.gotRequest) == 0 {
				th.gotRequest <- true
			}

			// maybe we just need to stop because we can't receive pushes anymore
			if stop {
				return
			}

			// continue to process more pushes, if any are remaining in the channel
			continue
		}
	}
}
