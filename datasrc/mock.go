package datasrc

import (
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

func (m *Mocksrc) Start()  {
	newRoot := func() core.Root {
		id := core.Root{}
		rand.Read(id[:])
		return id
	}
	i := Index(0)
	for {
		var parentNode *DagNode
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
			parentNode = layer.GetNodeAtDepth(d)
			if rand.Intn(2) == 0 {
				break
			}
		}
		ri := i
		if ri != 0 {
			if a := Index(rand.Intn(10)); a > i {
				ri = Index(rand.Intn(int(ri))) + 1
			}
			if ri < parentNode.Index {
				ri = parentNode.Index + 1
			}
		}
		parentKey := core.Root{}
		if parentNode != nil {
			parentKey = parentNode.Key
		}
		box := Box{
			Index: ri,
			Key: newRoot(),
			ParentKey: parentKey,
			Value: &MockBlock{},
		}
		fmt.Printf("add box: %d %x  parent: %x\n", ri, box.Key, box.ParentKey)
		m.Dag.AddBox(box)
		if a := Index(rand.Intn(10)); a & 3 == 3 {
			b := a >> 1
			i += b
		}
		time.Sleep(time.Millisecond * 10)
	}
}
