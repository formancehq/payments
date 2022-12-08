package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/app/payments"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNotFound = errors.New("not found")

type Store[TaskDescriptor payments.TaskDescriptor] interface {
	UpdateTaskStatus(ctx context.Context, provider string, descriptor TaskDescriptor,
		status payments.TaskStatus, err string) error
	FindTaskAndUpdateStatus(ctx context.Context, provider string, descriptor TaskDescriptor,
		status payments.TaskStatus, err string) (*payments.TaskState[TaskDescriptor], error)
	ListTaskStatesByStatus(ctx context.Context, provider string,
		status payments.TaskStatus) ([]payments.TaskState[TaskDescriptor], error)
	ListTaskStates(ctx context.Context, provider string) ([]payments.TaskState[TaskDescriptor], error)
	ReadOldestPendingTask(ctx context.Context, provider string) (*payments.TaskState[TaskDescriptor], error)
	ReadTaskState(ctx context.Context, provider string,
		descriptor TaskDescriptor) (*payments.TaskState[TaskDescriptor], error)
}

type InMemoryStore[TaskDescriptor payments.TaskDescriptor] struct {
	statuses map[string]payments.TaskStatus
	created  map[string]time.Time
	errors   map[string]string
}

func (s *InMemoryStore[TaskDescriptor]) ReadTaskState(ctx context.Context, provider string,
	descriptor TaskDescriptor,
) (*payments.TaskState[TaskDescriptor], error) {
	id := payments.IDFromDescriptor(descriptor)

	status, ok := s.statuses[id]
	if !ok {
		return nil, ErrNotFound
	}

	return &payments.TaskState[TaskDescriptor]{
		Provider:   provider,
		Descriptor: descriptor,
		Status:     status,
		Error:      s.errors[id],
		State:      nil,
		CreatedAt:  s.created[id],
	}, nil
}

func (s *InMemoryStore[TaskDescriptor]) ListTaskStates(ctx context.Context,
	provider string,
) ([]payments.TaskState[TaskDescriptor], error) {
	ret := make([]payments.TaskState[TaskDescriptor], 0)

	for id, status := range s.statuses {
		if !strings.HasPrefix(id, fmt.Sprintf("%s/", provider)) {
			continue
		}

		var descriptor TaskDescriptor

		payments.DescriptorFromID(id, &descriptor)

		ret = append(ret, payments.TaskState[TaskDescriptor]{
			Provider:   provider,
			Descriptor: descriptor,
			Status:     status,
			Error:      s.errors[id],
			State:      nil,
			CreatedAt:  s.created[id],
		})
	}

	return ret, nil
}

func (s *InMemoryStore[TaskDescriptor]) ReadOldestPendingTask(ctx context.Context,
	provider string,
) (*payments.TaskState[TaskDescriptor], error) {
	var (
		oldestDate time.Time
		oldestID   string
	)

	for id, status := range s.statuses {
		if status != payments.TaskStatusPending {
			continue
		}

		if oldestDate.IsZero() || s.created[id].Before(oldestDate) {
			oldestDate = s.created[id]
			oldestID = id
		}
	}

	if oldestDate.IsZero() {
		return nil, ErrNotFound
	}

	descriptorStr := strings.Split(oldestID, "/")[1]

	var descriptor TaskDescriptor

	payments.DescriptorFromID(descriptorStr, &descriptor)

	return &payments.TaskState[TaskDescriptor]{
		Provider:   provider,
		Descriptor: descriptor,
		Status:     payments.TaskStatusPending,
		State:      nil,
		CreatedAt:  s.created[oldestID],
	}, nil
}

func (s *InMemoryStore[TaskDescriptor]) ListTaskStatesByStatus(ctx context.Context,
	provider string, taskStatus payments.TaskStatus,
) ([]payments.TaskState[TaskDescriptor], error) {
	all, err := s.ListTaskStates(ctx, provider)
	if err != nil {
		return nil, err
	}

	ret := make([]payments.TaskState[TaskDescriptor], 0)

	for _, v := range all {
		if v.Status != taskStatus {
			continue
		}

		ret = append(ret, v)
	}

	return ret, nil
}

func (s *InMemoryStore[TaskDescriptor]) FindTaskAndUpdateStatus(ctx context.Context,
	provider string, descriptor TaskDescriptor, status payments.TaskStatus, taskErr string,
) (*payments.TaskState[TaskDescriptor], error) {
	err := s.UpdateTaskStatus(ctx, provider, descriptor, status, taskErr)
	if err != nil {
		return nil, err
	}

	return &payments.TaskState[TaskDescriptor]{
		Provider:   provider,
		Descriptor: descriptor,
		Status:     status,
		// CreatedAt:  s.created[fmt.Sprintf("%s/%s", provider, name)],
		Error: taskErr,
		State: nil,
	}, nil
}

func (s *InMemoryStore[TaskDescriptor]) UpdateTaskStatus(ctx context.Context, provider string,
	descriptor TaskDescriptor, status payments.TaskStatus, err string,
) error {
	taskID := payments.IDFromDescriptor(descriptor)
	key := fmt.Sprintf("%s/%s", provider, taskID)
	s.statuses[key] = status

	s.errors[key] = err
	if _, ok := s.created[key]; !ok {
		s.created[key] = time.Now()
	}

	return nil
}

func (s *InMemoryStore[TaskDescriptor]) Result(provider string,
	descriptor payments.TaskDescriptor,
) (payments.TaskStatus, string, bool) {
	taskID := payments.IDFromDescriptor(descriptor)
	key := fmt.Sprintf("%s/%s", provider, taskID)

	status, ok := s.statuses[key]
	if !ok {
		return "", "", false
	}

	return status, s.errors[key], true
}

func NewInMemoryStore[TaskDescriptor payments.TaskDescriptor]() *InMemoryStore[TaskDescriptor] {
	return &InMemoryStore[TaskDescriptor]{
		statuses: make(map[string]payments.TaskStatus),
		errors:   make(map[string]string),
		created:  make(map[string]time.Time),
	}
}

var _ Store[struct{}] = &InMemoryStore[struct{}]{}

type MongoDBStore[TaskDescriptor payments.TaskDescriptor] struct {
	db *mongo.Database
}

func (s *MongoDBStore[TaskDescriptor]) ReadTaskState(ctx context.Context, provider string,
	descriptor TaskDescriptor,
) (*payments.TaskState[TaskDescriptor], error) {
	ret := s.db.Collection(payments.TasksCollection).FindOne(ctx, map[string]any{
		"provider":   provider,
		"descriptor": descriptor,
	})
	if ret.Err() != nil {
		if errors.Is(ret.Err(), mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}

		return nil, ret.Err()
	}

	paymentState := payments.TaskState[TaskDescriptor]{}

	if err := ret.Decode(&paymentState); err != nil {
		return nil, err
	}

	return &paymentState, nil
}

func (s *MongoDBStore[TaskDescriptor]) ReadOldestPendingTask(ctx context.Context,
	provider string,
) (*payments.TaskState[TaskDescriptor], error) {
	ret := s.db.Collection(payments.TasksCollection).FindOne(ctx, map[string]any{
		"provider": provider,
		"status":   payments.TaskStatusPending,
	}, options.FindOne().SetSort(bson.M{"createdAt": 1}))
	if ret.Err() != nil {
		if errors.Is(ret.Err(), mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}

		return nil, ret.Err()
	}

	paymentState := &payments.TaskState[TaskDescriptor]{}

	if err := ret.Decode(paymentState); err != nil {
		return nil, err
	}

	return paymentState, nil
}

func (s *MongoDBStore[TaskDescriptor]) UpdateTaskStatus(ctx context.Context, provider string,
	descriptor TaskDescriptor, status payments.TaskStatus, taskErr string,
) error {
	_, err := s.db.Collection(payments.TasksCollection).UpdateOne(ctx, map[string]any{
		"provider":   provider,
		"descriptor": descriptor,
	}, map[string]any{
		"$set": map[string]any{
			"status": status,
			"error":  taskErr,
		},
		"$setOnInsert": map[string]any{
			"createdAt": time.Now(),
		},
	}, options.Update().SetUpsert(true))

	return err
}

func (s *MongoDBStore[TaskDescriptor]) FindTaskAndUpdateStatus(ctx context.Context, provider string,
	descriptor TaskDescriptor, status payments.TaskStatus, taskErr string,
) (*payments.TaskState[TaskDescriptor], error) {
	ret := s.db.Collection(payments.TasksCollection).FindOneAndUpdate(ctx, map[string]any{
		"provider":   provider,
		"descriptor": descriptor,
	}, map[string]any{
		"$set": map[string]any{
			"status": status,
			"error":  taskErr,
		},
		"$setOnInsert": map[string]any{
			"createdAt": time.Now(),
		},
	}, options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After))

	if ret.Err() != nil {
		return nil, errors.Wrap(ret.Err(), "retrieving task")
	}

	paymentState := &payments.TaskState[TaskDescriptor]{}

	if err := ret.Decode(paymentState); err != nil {
		return nil, errors.Wrap(err, "decoding task state")
	}

	return paymentState, nil
}

func (s *MongoDBStore[TaskDescriptor]) ListTaskStatesByStatus(ctx context.Context, provider string,
	status payments.TaskStatus,
) ([]payments.TaskState[TaskDescriptor], error) {
	cursor, err := s.db.Collection(payments.TasksCollection).Find(ctx, map[string]any{
		"provider": provider,
		"status":   status,
	})
	if err != nil {
		return nil, err
	}

	ret := make([]payments.TaskState[TaskDescriptor], 0)

	if err = cursor.All(ctx, &ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func (s *MongoDBStore[TaskDescriptor]) ListTaskStates(ctx context.Context,
	provider string,
) ([]payments.TaskState[TaskDescriptor], error) {
	cursor, err := s.db.Collection(payments.TasksCollection).Find(ctx, map[string]any{
		"provider": provider,
	})
	if err != nil {
		return nil, err
	}

	ret := make([]payments.TaskState[TaskDescriptor], 0)

	if err = cursor.All(ctx, &ret); err != nil {
		return nil, err
	}

	return ret, nil
}

var _ Store[struct{}] = &MongoDBStore[struct{}]{}

func NewMongoDBStore[TaskDescriptor payments.TaskDescriptor](db *mongo.Database) *MongoDBStore[TaskDescriptor] {
	return &MongoDBStore[TaskDescriptor]{db: db}
}
