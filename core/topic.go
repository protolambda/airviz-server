package core

type Topic byte

const (
	TopicDefault Topic = iota
	TopicBlocks
	TopicState
)