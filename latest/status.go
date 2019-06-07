package latest

import . "airviz/core"

type Status struct {
	Time   Index
	// length = length of cl
	Counts []uint32
	cl     *CycledLayers
}

func (stat *Status) UpdateStatus(start Index, counts []uint32) {
	// update local status
	stat.Time = start
	n := Index(len(stat.Counts))
	m := Index(len(counts))
	for i := Index(0); i < m; i++ {
		stat.Counts[(start+i)%n] = counts[i]
	}
}

type SyncDiffAtom struct {
	// depth in layer
	Depth uint32
	// node (has layer index)
	Node *Node
}

// note: client status is updated by mutating the status object
func (stat *Status) GetSyncDiff(start Index, end Index) ([]SyncDiffAtom, error) {
	length := stat.cl.length
	snapTime, layers := stat.cl.GetSnapshot()
	statLength := Index(len(stat.Counts))
	if end < snapTime {
		// request too old
		return nil, nil
	}
	if start > snapTime {
		// request too new
		return nil, nil
	}
	updates := make([]SyncDiffAtom, 0)
	dagStart := stat.cl.max
	if length > dagStart {
		dagStart = 0
	} else {
		dagStart -= length
	}
	// if out of scope, reset
	for i := start; i < stat.Time; i++ {
		statNorm := i % statLength
		stat.Counts[statNorm] = 0
	}
	for i := stat.Time + statLength; i < end; i++ {
		statNorm := i % statLength
		stat.Counts[statNorm] = 0
	}
	// check for lower bound
	if start < dagStart {
		start = dagStart
	}
	// check for upper bound
	if end > snapTime {
		end = snapTime
	}
	// compare data with status
	for i := start; i < end; i++ {
		iNorm := i % length
		statNorm := i % statLength
		prevCount := stat.Counts[statNorm]
		layer := layers[iNorm]
		if layer == nil {
			continue
		}
		currentCount := uint32(len(layer.nodes))
		for j := prevCount; j < currentCount; j++ {
			// we can update!
			updates = append(updates, SyncDiffAtom{
				Depth: j,
				Node: layer.nodes[j],
			})
		}
		stat.Counts[statNorm] = currentCount
	}
	return updates, nil
}
