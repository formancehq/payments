package activities

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

//go:generate mockgen -source publisher.go -destination publisher_generated.go -package activities . Publisher

// Publisher interface mirrors message.Publisher for mocking
type Publisher interface {
	Publish(topic string, messages ...*message.Message) error
	Close() error
}
