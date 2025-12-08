package activities_test

import (
	context "context"

	"github.com/ThreeDotsLabs/watermill/message"
)

type TestPublisher struct {
	channel chan *message.Message
}

func newTestPublisher() *TestPublisher {
	return &TestPublisher{
		channel: make(chan *message.Message, 100),
	}
}

func (p *TestPublisher) Publish(topic string, messages ...*message.Message) error {
	for _, message := range messages {
		p.channel <- message
	}
	return nil
}

func (p *TestPublisher) Close() error {
	close(p.channel)
	return nil
}

func (p *TestPublisher) Subscribe(ctx context.Context) <-chan *message.Message {
	return p.channel
}
