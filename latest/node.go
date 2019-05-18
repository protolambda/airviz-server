package latest

type DagNode struct {
	Box

	// parent can nil itself to decouple
	Parent **DagNode

	MyRef **DagNode

}

func NewDagNode(b Box) *DagNode {
	dn := &DagNode{
		Box: b,
		Parent: nil,
		MyRef: nil,
	}
	dn.MyRef = &dn
	return dn
}

func (dn *DagNode) Kill() {
	// nil the inner pointer, others won't be able to find this node anymore
	*dn.MyRef = nil
	dn.Parent = nil
}

