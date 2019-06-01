package datasrc

import (
	. "airviz/core"
	. "airviz/latest"
)

type DataEvent struct {
	Topic Topic
	Box *Box
}

type DataSrc interface {
	Start(evCh chan<- DataEvent)
}
