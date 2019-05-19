package latest

import . "airviz/core"

type Status struct {
	Time   Index
	// length = length of dag
	Counts []uint32
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
