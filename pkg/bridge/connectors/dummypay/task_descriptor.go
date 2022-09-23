package dummypay

type taskKey string

type TaskDescriptor struct {
	Key      taskKey
	FileName string
}

func (td TaskDescriptor) Is(key taskKey) bool {
	return td.Key == key
}
