package latest

import (
	"fmt"
	"github.com/protolambda/zrnt/eth2/core"
	"sync"
)

type DagLayer struct {

	sync.Mutex

	index Index

	nodes []*DagNode

}

func NewDagLayer(i Index) *DagLayer {
	return &DagLayer{
		index: i,
	}
}

func (dl *DagLayer) Index() Index {
	return dl.index
}

func (dl *DagLayer) Kill() {
	var wg sync.WaitGroup
	for _, n := range dl.nodes {
		wg.Add(1)
		go func() {
			n.Kill()
			wg.Done()
		}()
	}
	wg.Wait()
}

func (dl *DagLayer) AddNode(node *DagNode) {
	if node.Index != dl.index {
		panic(fmt.Sprintf("cannot add node with index %d layer with index %d", i, dl.index))
	}
	dl.Lock()
	dl.nodes = append(dl.nodes, node)
	dl.Unlock()
}

func (dl *DagLayer) GetNode(key core.Root) *DagNode {
	// ok w/ concurrency, layer is append only
	for i := 0; i < len(dl.nodes); i++ {
		if n := dl.nodes[i]; n.Key == key {
			return n
		}
	}
	return nil
}
