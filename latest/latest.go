package latest

import (
	. "airviz/core"
	"github.com/protolambda/zrnt/eth2/core"
	"sync"
)

type CycledLayers struct {

	sync.Mutex

	layers []*Layer

	// index lookup for each node in latest layers
	indices map[core.Root]Index

	// final, doesn't change
	length Index

	max Index

}

func NewLatestLayers(length Index) *CycledLayers {
	return &CycledLayers{
		layers: make([]*Layer, length, length),
		indices: make(map[core.Root]Index),
		length: length,
	}
}

func (cl *CycledLayers) Length() Index {
	return cl.length
}

// read box, asynchronously
func (cl *CycledLayers) Get(i Index) *Layer {
	if i > cl.max {
		return nil
	}
	if i + cl.length < cl.max {
		return nil
	}
	b := cl.layers[i % cl.length]
	if b == nil {
		return nil
	}
	// check if it's garbage from last cycle
	if b.index != i {
		return nil
	}
	return b
}

// Collect old boxes, save them, and forget about them locally.
func (cl *CycledLayers) GC() {
	// lock the latest data-structure,
	//  don't add anything while we're pruning away old data based on the current window
	cl.Lock()
	for i, layer := range cl.layers {
		if layer.index + cl.length < cl.max {
			// out of date, remove it
			cl.layers[i] = nil
		}
	}
	// clean up known indices and remove old floating-tasks
	for k, i := range cl.indices {
		if i + cl.length < cl.max {
			delete(cl.indices, k)
		}
	}
	// we're done with the latest data-structure, unlock it
	cl.Unlock()
}

// returns the time of the snapshot (end of cl), with a list of all the layers (shallow copy, items are append-only anyway)
func (cl *CycledLayers) GetSnapshot() (Index, []*Layer) {
	// make local copy of layer references, to compute status update on, without being affected by insertion of new layers
	cl.Lock()
	layers := append(make([]*Layer, 0, cl.length), cl.layers...)
	end := cl.max
	cl.Unlock()
	return end, layers
}

func (cl *CycledLayers) put(layer *Layer) {
	cl.layers[layer.index % cl.length] = layer
	if layer.index > cl.max {
		cl.max = layer.index
	}
}

func (cl *CycledLayers) AddBox(box *Node) {
	if box.ParentKey == box.Key {
		panic("cannot add box with parent key set to itself")
	}
	if box.Index + cl.length < cl.max {
		// box is too old to add
		return
	}
	cl.Lock()

	// Get the layer the node will be added to
	targetLayer := cl.Get(box.Index)
	if targetLayer == nil {
		targetLayer = NewLayer(box.Index)
		// create new layer
		cl.put(targetLayer)
	}

	// Add node to the graph
	targetLayer.AddNode(box)

	// Make note of the node index
	cl.indices[box.Key] = box.Index

	cl.Unlock()
}
