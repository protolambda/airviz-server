package latest

import (
	. "airviz/core"
	"fmt"
	"github.com/protolambda/zrnt/eth2/core"
	"sync"
)

type Layer struct {

	sync.Mutex

	index Index

	nodes []*Node

}

func NewLayer(i Index) *Layer {
	return &Layer{
		index: i,
	}
}

func (layer *Layer) AddNode(node *Node) {
	if node.Index != layer.index {
		panic(fmt.Sprintf("cannot add node with index %d layer with index %d", node.Index, layer.index))
	}
	layer.Lock()
	layer.nodes = append(layer.nodes, node)
	layer.Unlock()
}

func (layer *Layer) GetNode(key core.Root) *Node {
	// ok w/ concurrency, layer is append only
	for i := 0; i < len(layer.nodes); i++ {
		if n := layer.nodes[i]; n.Key == key {
			return n
		}
	}
	return nil
}

func (layer *Layer) GetNodeAtDepth(d uint32) *Node {
	if d < uint32(len(layer.nodes)) {
		return layer.nodes[d]
	} else {
		return nil
	}
}

func (layer *Layer) GetNodeDepth() uint32 {
	return uint32(len(layer.nodes))
}
