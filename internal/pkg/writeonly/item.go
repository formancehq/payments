package writeonly

type Item struct {
	Provider string `bson:"provider"`
	TaskID   string `bson:"taskID"`
	Data     any    `bson:"data"`
}
