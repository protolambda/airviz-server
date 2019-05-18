package latest

import "github.com/protolambda/zrnt/eth2/core"

type Serializable interface {
	Serialize() []byte
}

type Box struct {
	Index     Index
	Key       core.Root
	ParentKey core.Root
	Value     Serializable
}
