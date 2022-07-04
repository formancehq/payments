package writeonly

type Item struct {
	Provider string `bson:"provider"`
	TaskId   string `bson:"taskId"`
	Data     any    `bson:"data"`
}
