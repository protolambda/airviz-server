package datasrc

import "airviz/core"

type DataSrc interface {
	Start(triggerCh chan core.Trigger)
}
