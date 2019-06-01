package datasrc

import (
	. "airviz/core"
	. "airviz/latest"
	"fmt"
	"github.com/protolambda/zrnt/eth2/core"
	"math/rand"
	"time"
)

type Mocksrc struct {
}

type MockBlock struct {

}

func (mb *MockBlock) Serialize() []byte {
	return []byte("foobar")
}

func (m *Mocksrc) Start(triggerCh chan<- DataEvent)  {
	newRoot := func() core.Root {
		id := core.Root{}
		rand.Read(id[:])
		return id
	}
	lastRoot := core.Root{}
	lastIndex := Index(0)
	newBox := func() *Box {
		lastIndex += 1
		parentRoot := lastRoot
		lastRoot = newRoot()
		return &Box{
			Index: lastIndex,
			Key:lastRoot,
			ParentKey: parentRoot,
			Value: &MockBlock{},
		}
	}
	for {
		box := newBox()
		fmt.Printf("add box: %d %x  parent: %x\n", box.Index, box.Key, box.ParentKey)
		triggerCh <- DataEvent{Topic: TopicBlocks, Box: box}

		time.Sleep(time.Millisecond * 300)
	}
}
