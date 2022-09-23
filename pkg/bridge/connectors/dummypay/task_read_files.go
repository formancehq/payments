package dummypay

import (
	"context"
	"os"
	"time"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/task"
)

const taskKeyReadFiles = "read-files"

func newTaskReadFiles() TaskDescriptor {
	return TaskDescriptor{
		Key: taskKeyReadFiles,
	}
}

func taskReadFiles(config Config) task.Task {
	return func(ctx context.Context, logger sharedlogging.Logger,
		scheduler task.Scheduler[TaskDescriptor]) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(10 * time.Second): // Could be configurable using Config object
				logger.Infof("Opening directory '%s'...", config.Directory)
				dir, err := os.ReadDir(config.Directory)
				if err != nil {
					logger.Errorf("Error opening directory '%s': %s", config.Directory, err)
					continue
				}

				logger.Infof("Found %d files", len(dir))
				for _, file := range dir {
					err = scheduler.Schedule(newTaskIngest(file.Name()), true)
					if err != nil {
						logger.Errorf("Error scheduling task '%s': %s", file.Name(), err)
						continue
					}
				}
			}
		}
	}
}
