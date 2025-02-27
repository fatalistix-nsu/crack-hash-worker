package model

type Task struct {
	RequestId string
	TaskId    string
	Alphabet  string
	Hash      string
	MaxLength uint64
	Start     uint64
	End       uint64
}
