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
	Dag *Dag
}

type MockBlock struct {

}

func (mb *MockBlock) Serialize() []byte {
	return []byte("foobar")
}

func (m *Mocksrc) Start(triggerCh chan Trigger)  {
	newRoot := func() core.Root {
		id := core.Root{}
		rand.Read(id[:])
		return id
	}
	for {
		var node *DagNode
		snapTime, snap := m.Dag.GetSnapshot()
		start := Index(0)
		if snapTime > Index(len(snap)) {
			start = snapTime - Index(len(snap))
		}
		for pl := int64(snapTime); pl >= int64(start); pl-- {
			t := pl % int64(len(snap))
			layer := snap[t]
			if layer == nil {
				continue
			}
			d := layer.GetNodeDepth()
			if d == 0 {
				continue
			}
			d = uint32(rand.Intn(int(d)))
			node = layer.GetNodeAtDepth(d)
			break
		}
		for node != nil && node.MyRef != nil && (*node.MyRef).Parent != nil {
			if rand.Intn(10) > 7 {
				p := *(*node.MyRef).Parent
				if p != nil {
					node = p
				} else {
					break
				}
			} else {
				break
			}
		}
		parentKey := core.Root{}
		ri := Index(0)
		if node != nil {
			parentKey = node.Key
			ri = node.Index + Index(rand.Intn(3))
		}
		box := Box{
			Index: ri,
			Key: newRoot(),
			ParentKey: parentKey,
			Value: &MockBlock{},
		}
		fmt.Printf("add box: %d %x  parent: %x\n", ri, box.Key, box.ParentKey)
		m.Dag.AddBox(box)
		triggerCh <- Trigger{Topic: TopicBlocks, Index: box.Index}
		time.Sleep(time.Millisecond * 300)
	}
}
