package latest

import (
	. "airviz/core"
	"github.com/protolambda/zrnt/eth2/core"
)

type Serializable interface {
	Serialize() []byte
}

type Node struct {
	Index     Index
	Key       core.Root
	ParentKey core.Root
	Value     Serializable
}
