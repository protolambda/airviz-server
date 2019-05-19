package latest

import (
	. "airviz/core"
	"errors"
	"github.com/protolambda/zrnt/eth2/core"
	"sync"
)

type KeyFunc func(box Box, other Box)

// The "DAG". Just a big directed tree, with floating components, to be connected later to the root of the tree
type Dag struct {

	sync.Mutex

	// nodes without a parent, but expected to have one within latest range
	// parent-key -> node key -> node
	floating map[core.Root]map[core.Root]*DagNode
	// index lookup for each node in latest layers
	indices map[core.Root]Index
	// layers, for latest N indices
	layers *LatestLayers

}

func NewDag(length Index) *Dag {
	return &Dag{
		floating: make(map[core.Root]map[core.Root]*DagNode),
		indices: make(map[core.Root]Index),
		layers: NewLatestLayers(length),
	}
}

func (dag *Dag) GetEmptyStatus() *Status {
	return &Status{0, make([]uint32, dag.layers.length)}
}

type StatusUpdateAtom struct {
	// depth in layer
	Depth uint32
	// node (has layer index)
	Node *DagNode
}

// returns the time of the snapshot (end of dag), with a list of all the layers (shallow copy, items are append-only anyway)
func (dag *Dag) GetSnapshot() (Index, []*DagLayer) {
	length := dag.layers.length
	// make local copy of layer references, to compute status update on, without being affected by insertion of new layers
	dag.Lock()
	layers := make([]*DagLayer, 0, length)
	layers = append(layers, dag.layers.layers...)
	end := dag.layers.max
	dag.Unlock()
	return end, layers
}

// note: client status is updated by mutating the given status object
func (dag *Dag) GetStatusUpdate(stat *Status, start Index, end Index) ([]StatusUpdateAtom, error) {
	length := dag.layers.length
	snapTime, layers := dag.GetSnapshot()
	srcLength := Index(len(stat.Counts))
	if srcLength != length {
		return nil, errors.New("cannot compute status update for client status with different length")
	}
	if end + length < snapTime {
		return nil, errors.New("requested client time is too long ago, data is not maintained")
	}
	updates := make([]StatusUpdateAtom, 0)
	dagStart := dag.layers.max
	if length > dagStart {
		dagStart = 0
	} else {
		dagStart -= length
	}
	// check for lower bound
	if start < dagStart {
		start = dagStart
	}
	// check for upper bound
	if end > snapTime {
		end = snapTime
	}
	if stat.Time < dagStart {
		// client was lagging behind, reset counters in outdated data
		diff := dagStart - stat.Time
		// you can only get so much out of date, don't repeat resets
		if diff > length {
			diff = length
		}
		s := stat.Time % length
		e := s + diff
		for i := s; i < e; i++ {
			stat.Counts[i % length] = 0
		}
	}
	for i := start; i < end; i++ {
		iNorm := i % length
		prevCount := stat.Counts[iNorm]
		layer := layers[iNorm]
		currentCount := uint32(len(layer.nodes))
		for j := prevCount; j < currentCount; j++ {
			// we can update!
			updates = append(updates, StatusUpdateAtom{
				Depth: j,
				Node: layer.nodes[j],
			})
		}
		stat.Counts[iNorm] = currentCount
	}
	return updates, nil
}

func (dag *Dag) AddBox(box Box) {
	if box.ParentKey == box.Key {
		panic("cannot add box with parent key set to itself")
	}
	if box.Index+ dag.layers.length <= dag.layers.max {
		// box is too old to add
		return
	}
	parentIndex, hasParent := dag.indices[box.ParentKey]
	dag.Lock()

	// Find parent of the node
	var parentNode **DagNode
	if hasParent {
		parentLayer := dag.layers.Get(parentIndex)
		if parentLayer != nil {
			p := parentLayer.GetNode(box.ParentKey)
			if p != nil {
				parentNode = p.MyRef
			}
		}
	}
	// Get the layer the node will be added to
	targetLayer := dag.layers.Get(box.Index)
	if targetLayer == nil {
		targetLayer = NewDagLayer(box.Index)
		// create new layer
		dag.layers.Put(targetLayer)
	}

	// Create the node
	node := NewDagNode(box)

	// Connect parent to child, if any
	if parentNode == nil {
		// parent could be within latest range, we'll keep this node floating around and wait for the parent to arrive
		floating, hasSiblings := dag.floating[box.ParentKey]
		if hasSiblings {
			floating[node.Key] = node
		} else {
			dag.floating[box.ParentKey] = map[core.Root]*DagNode{
				node.Key: node,
			}
		}
	} else {
		// connect parent to node
		node.Parent = parentNode
	}

	// We may have children waiting for this node, check
	floating, hasChildren := dag.floating[box.Key]
	if hasChildren {
		for _, child := range floating {
			child.Parent = node.MyRef
		}
	}

	// Add node to the graph
	targetLayer.AddNode(node)

	// Make note of the node index
	dag.indices[box.Key] = box.Index

	dag.Unlock()
}

func (dag *Dag) GC() {
	dag.Lock()
	// clean up known indices and remove old floating-tasks
	for k, i := range dag.indices {
		if i + dag.layers.length < dag.layers.max {
			delete(dag.indices, k)
			delete(dag.floating, k)
		}
	}
	dag.Unlock()
	dag.layers.GC()
}
