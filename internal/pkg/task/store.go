package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	payments2 "github.com/numary/payments/internal/pkg/payments"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNotFound = errors.New("not found")

type Store[TaskDescriptor payments2.TaskDescriptor] interface {
	UpdateTaskStatus(ctx context.Context, provider string, descriptor TaskDescriptor, status payments2.TaskStatus, err string) error
	FindTaskAndUpdateStatus(ctx context.Context, provider string, descriptor TaskDescriptor,
		status payments2.TaskStatus, err string) (*payments2.TaskState[TaskDescriptor], error)
	ListTaskStatesByStatus(ctx context.Context, provider string, status payments2.TaskStatus) ([]payments2.TaskState[TaskDescriptor], error)
	ListTaskStates(ctx context.Context, provider string) ([]payments2.TaskState[TaskDescriptor], error)
	ReadOldestPendingTask(ctx context.Context, provider string) (*payments2.TaskState[TaskDescriptor], error)
	ReadTaskState(ctx context.Context, provider string, descriptor TaskDescriptor) (*payments2.TaskState[TaskDescriptor], error)
}

type inMemoryStore[TaskDescriptor payments2.TaskDescriptor] struct {
	statuses map[string]payments2.TaskStatus
	created  map[string]time.Time
	errors   map[string]string
}

func (s *inMemoryStore[TaskDescriptor]) ReadTaskState(ctx context.Context, provider string, descriptor TaskDescriptor) (*payments2.TaskState[TaskDescriptor], error) {
	id := payments2.IDFromDescriptor(descriptor)
	status, ok := s.statuses[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &payments2.TaskState[TaskDescriptor]{
		Provider:   provider,
		Descriptor: descriptor,
		Status:     status,
		Error:      s.errors[id],
		State:      nil,
		CreatedAt:  s.created[id],
	}, nil
}

func (s *inMemoryStore[TaskDescriptor]) ListTaskStates(ctx context.Context, provider string) ([]payments2.TaskState[TaskDescriptor], error) {
	ret := make([]payments2.TaskState[TaskDescriptor], 0)
	for id, status := range s.statuses {
		if !strings.HasPrefix(id, fmt.Sprintf("%s/", provider)) {
			continue
		}

		var descriptor TaskDescriptor
		payments2.DescriptorFromID(id, &descriptor)

		ret = append(ret, payments2.TaskState[TaskDescriptor]{
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

func (s *inMemoryStore[TaskDescriptor]) ReadOldestPendingTask(ctx context.Context, provider string) (*payments2.TaskState[TaskDescriptor], error) {
	var (
		oldestDate time.Time
		oldestId   string
	)
	for id, status := range s.statuses {
		if status != payments2.TaskStatusPending {
			continue
		}
		if oldestDate.IsZero() || s.created[id].Before(oldestDate) {
			oldestDate = s.created[id]
			oldestId = id
		}
	}
	if oldestDate.IsZero() {
		return nil, ErrNotFound
	}

	descriptorStr := strings.Split(oldestId, "/")[1]

	var descriptor TaskDescriptor
	payments2.DescriptorFromID(descriptorStr, &descriptor)

	return &payments2.TaskState[TaskDescriptor]{
		Provider:   provider,
		Descriptor: descriptor,
		Status:     payments2.TaskStatusPending,
		State:      nil,
		CreatedAt:  s.created[oldestId],
	}, nil
}

func (s *inMemoryStore[TaskDescriptor]) ListTaskStatesByStatus(ctx context.Context, provider string, taskStatus payments2.TaskStatus) ([]payments2.TaskState[TaskDescriptor], error) {
	all, err := s.ListTaskStates(ctx, provider)
	if err != nil {
		return nil, err
	}

	ret := make([]payments2.TaskState[TaskDescriptor], 0)
	for _, v := range all {
		if v.Status != taskStatus {
			continue
		}
		ret = append(ret, v)
	}

	return ret, nil
}

func (s *inMemoryStore[TaskDescriptor]) FindTaskAndUpdateStatus(ctx context.Context, provider string, descriptor TaskDescriptor, status payments2.TaskStatus, taskErr string) (*payments2.TaskState[TaskDescriptor], error) {
	err := s.UpdateTaskStatus(ctx, provider, descriptor, status, taskErr)
	if err != nil {
		return nil, err
	}

	return &payments2.TaskState[TaskDescriptor]{
		Provider:   provider,
		Descriptor: descriptor,
		Status:     status,
		// CreatedAt:  s.created[fmt.Sprintf("%s/%s", provider, name)],
		Error: taskErr,
		State: nil,
	}, nil
}

func (s *inMemoryStore[TaskDescriptor]) UpdateTaskStatus(ctx context.Context, provider string, descriptor TaskDescriptor, status payments2.TaskStatus, err string) error {
	taskId := payments2.IDFromDescriptor(descriptor)
	key := fmt.Sprintf("%s/%s", provider, taskId)
	s.statuses[key] = status
	s.errors[key] = err
	if _, ok := s.created[key]; !ok {
		s.created[key] = time.Now()
	}
	return nil
}

func (s *inMemoryStore[TaskDescriptor]) Result(provider string, descriptor payments2.TaskDescriptor) (payments2.TaskStatus, string, bool) {
	taskId := payments2.IDFromDescriptor(descriptor)
	key := fmt.Sprintf("%s/%s", provider, taskId)
	status, ok := s.statuses[key]
	if !ok {
		return "", "", false
	}
	return status, s.errors[key], true
}

func NewInMemoryStore[TaskDescriptor payments2.TaskDescriptor]() *inMemoryStore[TaskDescriptor] {
	return &inMemoryStore[TaskDescriptor]{
		statuses: make(map[string]payments2.TaskStatus),
		errors:   make(map[string]string),
		created:  make(map[string]time.Time),
	}
}

var _ Store[struct{}] = &inMemoryStore[struct{}]{}

type mongoDBStore[TaskDescriptor payments2.TaskDescriptor] struct {
	db *mongo.Database
}

func (m *mongoDBStore[TaskDescriptor]) ReadTaskState(ctx context.Context, provider string, descriptor TaskDescriptor) (*payments2.TaskState[TaskDescriptor], error) {
	ret := m.db.Collection(payments2.TasksCollection).FindOne(ctx, map[string]any{
		"provider":   provider,
		"descriptor": descriptor,
	})
	if ret.Err() != nil {
		if ret.Err() == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, ret.Err()
	}
	ts := payments2.TaskState[TaskDescriptor]{}
	err := ret.Decode(&ts)
	if err != nil {
		return nil, err
	}

	return &ts, nil
}

func (m *mongoDBStore[TaskDescriptor]) ReadOldestPendingTask(ctx context.Context, provider string) (*payments2.TaskState[TaskDescriptor], error) {
	ret := m.db.Collection(payments2.TasksCollection).FindOne(ctx, map[string]any{
		"provider": provider,
		"status":   payments2.TaskStatusPending,
	}, options.FindOne().SetSort(bson.M{"createdAt": 1}))
	if ret.Err() != nil {
		if ret.Err() == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, ret.Err()
	}
	ps := &payments2.TaskState[TaskDescriptor]{}
	err := ret.Decode(ps)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

func (m *mongoDBStore[TaskDescriptor]) UpdateTaskStatus(ctx context.Context, provider string, descriptor TaskDescriptor, status payments2.TaskStatus, taskErr string) error {
	_, err := m.db.Collection(payments2.TasksCollection).UpdateOne(ctx, map[string]any{
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

func (m *mongoDBStore[TaskDescriptor]) FindTaskAndUpdateStatus(ctx context.Context, provider string, descriptor TaskDescriptor, status payments2.TaskStatus, taskErr string) (*payments2.TaskState[TaskDescriptor], error) {
	ret := m.db.Collection(payments2.TasksCollection).FindOneAndUpdate(ctx, map[string]any{
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
	ps := &payments2.TaskState[TaskDescriptor]{}
	err := ret.Decode(ps)
	if err != nil {
		return nil, errors.Wrap(err, "decoding task state")
	}
	return ps, nil
}

func (m *mongoDBStore[TaskDescriptor]) ListTaskStatesByStatus(ctx context.Context, provider string, status payments2.TaskStatus) ([]payments2.TaskState[TaskDescriptor], error) {
	cursor, err := m.db.Collection(payments2.TasksCollection).Find(ctx, map[string]any{
		"provider": provider,
		"status":   status,
	})
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	ret := make([]payments2.TaskState[TaskDescriptor], 0)
	err = cursor.All(ctx, &ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (m *mongoDBStore[TaskDescriptor]) ListTaskStates(ctx context.Context, provider string) ([]payments2.TaskState[TaskDescriptor], error) {
	cursor, err := m.db.Collection(payments2.TasksCollection).Find(ctx, map[string]any{
		"provider": provider,
	})
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	ret := make([]payments2.TaskState[TaskDescriptor], 0)
	err = cursor.All(ctx, &ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

var _ Store[struct{}] = &mongoDBStore[struct{}]{}

func NewMongoDBStore[TaskDescriptor payments2.TaskDescriptor](db *mongo.Database) *mongoDBStore[TaskDescriptor] {
	return &mongoDBStore[TaskDescriptor]{db: db}
}
