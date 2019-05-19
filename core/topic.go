package core

type Topic uint32

const (
	TopicDefault Topic = iota
	TopicBlocks
	TopicState
)