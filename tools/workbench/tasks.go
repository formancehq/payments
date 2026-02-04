package workbench

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

// taskTypeNames maps TaskType to human-readable strings.
var taskTypeNames = map[models.TaskType]string{
	models.TASK_FETCH_OTHERS:            "FETCH_OTHERS",
	models.TASK_FETCH_ACCOUNTS:          "FETCH_ACCOUNTS",
	models.TASK_FETCH_BALANCES:          "FETCH_BALANCES",
	models.TASK_FETCH_EXTERNAL_ACCOUNTS: "FETCH_EXTERNAL_ACCOUNTS",
	models.TASK_FETCH_PAYMENTS:          "FETCH_PAYMENTS",
	models.TASK_CREATE_WEBHOOKS:         "CREATE_WEBHOOKS",
}

// taskTypeName returns the string name for a TaskType.
func taskTypeName(t models.TaskType) string {
	if name, ok := taskTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN_%d", int(t))
}

// TaskStatus represents the status of a task.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusSkipped   TaskStatus = "skipped"
)

// TaskNode represents a task in the execution tree.
type TaskNode struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Name        string          `json:"name,omitempty"`
	Status      TaskStatus      `json:"status"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Duration    time.Duration   `json:"duration,omitempty"`
	Error       string          `json:"error,omitempty"`
	ItemsCount  int             `json:"items_count,omitempty"`
	FromPayload json.RawMessage `json:"from_payload,omitempty"`
	Children    []*TaskNode     `json:"children,omitempty"`
	Executions  []TaskExecution `json:"executions,omitempty"`
}

// TaskExecution represents a single execution of a task.
type TaskExecution struct {
	ID          string        `json:"id"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	Status      TaskStatus    `json:"status"`
	Error       string        `json:"error,omitempty"`
	ItemsCount  int           `json:"items_count"`
	PageNumber  int           `json:"page_number"`
	HasMore     bool          `json:"has_more"`
}

// TaskTracker tracks task execution for the workbench.
type TaskTracker struct {
	mu sync.RWMutex

	// The root task tree (from Install)
	rootTree []*TaskNode

	// Flat list of all executions for history
	executions []TaskExecution

	// Currently running task
	currentTask *TaskNode

	// Execution mode
	stepMode bool // If true, pause after each task

	// Channels for step control
	stepChan chan struct{}
	stopChan chan struct{}

	// Stats
	stats TaskStats
}

// TaskStats holds execution statistics.
type TaskStats struct {
	TotalExecutions   int           `json:"total_executions"`
	SuccessCount      int           `json:"success_count"`
	FailureCount      int           `json:"failure_count"`
	TotalItemsFetched int           `json:"total_items_fetched"`
	TotalDuration     time.Duration `json:"total_duration"`
	LastExecutionAt   *time.Time    `json:"last_execution_at,omitempty"`
}

// NewTaskTracker creates a new task tracker.
func NewTaskTracker() *TaskTracker {
	return &TaskTracker{
		executions: make([]TaskExecution, 0),
		stepChan:   make(chan struct{}, 1),
		stopChan:   make(chan struct{}),
	}
}

// SetTaskTree sets the task tree from the connector's Install response.
func (t *TaskTracker) SetTaskTree(tree models.ConnectorTasksTree) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.rootTree = t.convertTree(tree, nil)
}

func (t *TaskTracker) convertTree(tree models.ConnectorTasksTree, fromPayload json.RawMessage) []*TaskNode {
	var nodes []*TaskNode
	for _, task := range tree {
		node := &TaskNode{
			ID:          uuid.New().String(),
			Type:        taskTypeName(task.TaskType),
			Name:        task.Name,
			Status:      TaskStatusPending,
			FromPayload: fromPayload,
			Executions:  make([]TaskExecution, 0),
		}
		if len(task.NextTasks) > 0 {
			// Children will be created dynamically based on fetched items
			// For now, just note that there are child tasks
			node.Children = make([]*TaskNode, 0)
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// GetTaskTree returns the current task tree.
func (t *TaskTracker) GetTaskTree() []*TaskNode {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.rootTree
}

// GetExecutions returns recent executions.
func (t *TaskTracker) GetExecutions(limit int) []TaskExecution {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if limit <= 0 || limit > len(t.executions) {
		limit = len(t.executions)
	}

	// Return most recent first
	result := make([]TaskExecution, limit)
	for i := 0; i < limit; i++ {
		result[i] = t.executions[len(t.executions)-1-i]
	}
	return result
}

// GetCurrentTask returns the currently running task.
func (t *TaskTracker) GetCurrentTask() *TaskNode {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.currentTask
}

// GetStats returns execution statistics.
func (t *TaskTracker) GetStats() TaskStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.stats
}

// StartTask marks a task as started.
func (t *TaskTracker) StartTask(taskType models.TaskType, name string, fromPayload json.RawMessage) *TaskExecution {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	exec := TaskExecution{
		ID:        uuid.New().String(),
		StartedAt: now,
		Status:    TaskStatusRunning,
	}

	// Find or create the task node
	taskTypStr := taskTypeName(taskType)
	node := t.findOrCreateNode(taskTypStr, name, fromPayload)
	if node != nil {
		node.Status = TaskStatusRunning
		node.StartedAt = &now
		t.currentTask = node
	}

	return &exec
}

// CompleteTask marks a task as completed.
func (t *TaskTracker) CompleteTask(exec *TaskExecution, itemsCount int, hasMore bool, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	exec.CompletedAt = &now
	exec.Duration = now.Sub(exec.StartedAt)
	exec.ItemsCount = itemsCount
	exec.HasMore = hasMore

	if err != nil {
		exec.Status = TaskStatusFailed
		exec.Error = err.Error()
		t.stats.FailureCount++
	} else {
		exec.Status = TaskStatusCompleted
		t.stats.SuccessCount++
	}

	t.stats.TotalExecutions++
	t.stats.TotalItemsFetched += itemsCount
	t.stats.TotalDuration += exec.Duration
	t.stats.LastExecutionAt = &now

	// Add to history
	t.executions = append(t.executions, *exec)
	if len(t.executions) > 1000 {
		t.executions = t.executions[1:]
	}

	// Update node
	if t.currentTask != nil {
		t.currentTask.Status = exec.Status
		t.currentTask.CompletedAt = &now
		t.currentTask.Duration = exec.Duration
		t.currentTask.ItemsCount += itemsCount
		if err != nil {
			t.currentTask.Error = err.Error()
		}
		t.currentTask.Executions = append(t.currentTask.Executions, *exec)
		t.currentTask = nil
	}
}

func (t *TaskTracker) findOrCreateNode(taskType string, name string, fromPayload json.RawMessage) *TaskNode {
	payloadKey := string(fromPayload)
	
	// Recursive search function
	var findInNodes func([]*TaskNode) *TaskNode
	findInNodes = func(nodes []*TaskNode) *TaskNode {
		for _, node := range nodes {
			// Match by type and payload (payload is the key for child tasks)
			if node.Type == taskType {
				// For child tasks, match by fromPayload
				if payloadKey != "" && string(node.FromPayload) == payloadKey {
					return node
				}
				// For root tasks with no payload, match by name or just type
				if payloadKey == "" && (name == "" || node.Name == name) {
					return node
				}
			}
			// Search children recursively
			if len(node.Children) > 0 {
				if found := findInNodes(node.Children); found != nil {
					return found
				}
			}
		}
		return nil
	}

	// Search existing tree
	if found := findInNodes(t.rootTree); found != nil {
		return found
	}

	// Create new node at root if not found (shouldn't happen for child tasks)
	node := &TaskNode{
		ID:          uuid.New().String(),
		Type:        taskType,
		Name:        name,
		Status:      TaskStatusPending,
		FromPayload: fromPayload,
		Executions:  make([]TaskExecution, 0),
	}
	t.rootTree = append(t.rootTree, node)
	return node
}

// AddChildTask adds a child task node (for tasks spawned by fetched items).
func (t *TaskTracker) AddChildTask(parentType models.TaskType, childType models.TaskType, childName string, fromPayload json.RawMessage) *TaskNode {
	t.mu.Lock()
	defer t.mu.Unlock()

	parentTypStr := taskTypeName(parentType)
	childTypStr := taskTypeName(childType)

	// Find parent
	var parent *TaskNode
	for _, node := range t.rootTree {
		if node.Type == parentTypStr {
			parent = node
			break
		}
	}

	if parent == nil {
		return nil
	}

	// Check if child already exists
	payloadKey := string(fromPayload)
	for _, child := range parent.Children {
		if child.Type == childTypStr && string(child.FromPayload) == payloadKey {
			return child
		}
	}

	// Create child node
	child := &TaskNode{
		ID:          uuid.New().String(),
		Type:        childTypStr,
		Name:        childName,
		Status:      TaskStatusPending,
		FromPayload: fromPayload,
		Executions:  make([]TaskExecution, 0),
		Children:    make([]*TaskNode, 0),
	}
	parent.Children = append(parent.Children, child)

	return child
}

// SetStepMode enables or disables step-by-step execution.
func (t *TaskTracker) SetStepMode(enabled bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stepMode = enabled
}

// IsStepMode returns whether step mode is enabled.
func (t *TaskTracker) IsStepMode() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.stepMode
}

// Step signals to continue to the next task (when in step mode).
func (t *TaskTracker) Step() {
	select {
	case t.stepChan <- struct{}{}:
	default:
	}
}

// WaitForStep waits for a step signal (when in step mode).
func (t *TaskTracker) WaitForStep() bool {
	if !t.IsStepMode() {
		return true
	}

	select {
	case <-t.stepChan:
		return true
	case <-t.stopChan:
		return false
	}
}

// Reset resets the task tracker.
func (t *TaskTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Reset all node statuses
	var resetNodes func([]*TaskNode)
	resetNodes = func(nodes []*TaskNode) {
		for _, node := range nodes {
			node.Status = TaskStatusPending
			node.StartedAt = nil
			node.CompletedAt = nil
			node.Duration = 0
			node.Error = ""
			node.ItemsCount = 0
			node.Executions = make([]TaskExecution, 0)
			if len(node.Children) > 0 {
				resetNodes(node.Children)
			}
		}
	}
	resetNodes(t.rootTree)

	t.executions = make([]TaskExecution, 0)
	t.currentTask = nil
	t.stats = TaskStats{}
}

// TaskTreeSummary provides a summary view of the task tree.
type TaskTreeSummary struct {
	Tree         []*TaskNodeSummary `json:"tree"`
	CurrentTask  *TaskNodeSummary   `json:"current_task,omitempty"`
	Stats        TaskStats          `json:"stats"`
	StepMode     bool               `json:"step_mode"`
	IsRunning    bool               `json:"is_running"`
}

// TaskNodeSummary is a simplified task node for the UI.
type TaskNodeSummary struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Name         string             `json:"name,omitempty"`
	Status       TaskStatus         `json:"status"`
	Duration     string             `json:"duration,omitempty"`
	ItemsCount   int                `json:"items_count"`
	Error        string             `json:"error,omitempty"`
	Children     []*TaskNodeSummary `json:"children,omitempty"`
	ChildCount   int                `json:"child_count"`
	LastExecTime string             `json:"last_exec_time,omitempty"`
}

// GetSummary returns a summary of the task tree.
func (t *TaskTracker) GetSummary() TaskTreeSummary {
	t.mu.RLock()
	defer t.mu.RUnlock()

	summary := TaskTreeSummary{
		Tree:      t.summarizeNodes(t.rootTree),
		Stats:     t.stats,
		StepMode:  t.stepMode,
		IsRunning: t.currentTask != nil,
	}

	if t.currentTask != nil {
		summary.CurrentTask = t.summarizeNode(t.currentTask)
	}

	return summary
}

func (t *TaskTracker) summarizeNodes(nodes []*TaskNode) []*TaskNodeSummary {
	var result []*TaskNodeSummary
	for _, node := range nodes {
		result = append(result, t.summarizeNode(node))
	}
	return result
}

func (t *TaskTracker) summarizeNode(node *TaskNode) *TaskNodeSummary {
	s := &TaskNodeSummary{
		ID:         node.ID,
		Type:       node.Type,
		Name:       node.Name,
		Status:     node.Status,
		ItemsCount: node.ItemsCount,
		Error:      node.Error,
		ChildCount: len(node.Children),
	}

	if node.Duration > 0 {
		s.Duration = fmt.Sprintf("%.2fs", node.Duration.Seconds())
	}

	if node.CompletedAt != nil {
		s.LastExecTime = node.CompletedAt.Format("15:04:05")
	}

	// Include children (limit depth to avoid huge payloads)
	if len(node.Children) > 0 && len(node.Children) <= 50 {
		s.Children = t.summarizeNodes(node.Children)
	}

	return s
}
