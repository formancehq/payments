package core

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/gibson042/canonicaljson-go"
	"go.mongodb.org/mongo-driver/bson"
)

type TaskStatus string

var (
	TaskStatusStopped    TaskStatus = "stopped"
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusActive     TaskStatus = "active"
	TaskStatusTerminated TaskStatus = "terminated"
	TaskStatusFailed     TaskStatus = "failed"
)

type TaskState[Descriptor TaskDescriptor] struct {
	Provider   string     `json:"provider" bson:"provider"`
	Descriptor Descriptor `json:"descriptor" bson:"descriptor"`
	CreatedAt  time.Time  `json:"createdAt" bson:"createdAt"`
	Status     TaskStatus `json:"status" bson:"status"`
	Error      string     `json:"error" bson:"error"`
	State      bson.Raw   `json:"state" bson:"state"`
}

type taskState[Descriptor any] TaskState[Descriptor]

func (t TaskState[Descriptor]) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		taskState[Descriptor]
		ID string `json:"id"`
	}{
		taskState: taskState[Descriptor](t),
		ID:        IDFromDescriptor(t.Descriptor),
	})
}

type TaskDescriptor any

func DescriptorFromID(id string, to interface{}) {
	data, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		panic(err)
	}
	err = canonicaljson.Unmarshal(data, to)
	if err != nil {
		panic(err)
	}
}

func IDFromDescriptor(d TaskDescriptor) string {
	data, err := canonicaljson.Marshal(d)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(data)
}
