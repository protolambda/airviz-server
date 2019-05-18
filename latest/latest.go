package latest

import (
	"sync"
)

type LatestLayers struct {

	sync.Mutex

	layers []*DagLayer

	// final, doesn't change
	length Index

	max Index

}

func NewLatestLayers(length Index) *LatestLayers {
	return &LatestLayers{
		layers: make([]*DagLayer, length, length),
		length: length,
	}
}

func (ll *LatestLayers) Length() Index {
	return ll.length
}

// add new layer, synchronously
func (ll *LatestLayers) Put(layer *DagLayer) {
	ll.Lock()
	i := layer.Index()
	ll.layers[i % ll.length] = layer
	if i > ll.max {
		ll.max = i
	}
	ll.Unlock()
}

// read box, asynchronously
func (ll *LatestLayers) Get(i Index) *DagLayer {
	if i > ll.max {
		return nil
	}
	if i < ll.max - ll.length {
		return nil
	}
	b := ll.layers[i % ll.length]
	if b == nil {
		return nil
	}
	bi := b.Index()
	// check if it's garbage from last cycle
	if bi != i {
		return nil
	}
	return b
}

// Collect old boxes, save them, and forget about them locally.
func (ll *LatestLayers) GC() {
	// lock the latest data-structure,
	//  don't add anything while we're pruning away old data based on the current window
	// The pruning is fast, and not blocked by IO.
	ll.Lock()
	var wg sync.WaitGroup
	for i, layer := range ll.layers {
		if layer.index + ll.length < ll.max {
			// out of date, remove it
			ll.layers[i] = nil
			wg.Add(1)
			go func() {
				layer.Kill()
				wg.Done()
			}()
		}
	}
	// we're done with the latest data-structure, unlock it
	ll.Unlock()
	// wait for saver to complete
	wg.Wait()
}
