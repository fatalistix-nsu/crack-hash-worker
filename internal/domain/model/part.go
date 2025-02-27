package model

type Part struct {
	RequestId string
	TaskId    string
	Alphabet  string
	Hash      string
	MaxLength uint64
	Start     uint64
	End       uint64
}

type CompletedPart struct {
	RequestId string
	TaskId    string
	Data      []string
	Start     uint64
	End       uint64
}
