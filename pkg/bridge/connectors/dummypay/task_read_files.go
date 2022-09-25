package dummypay

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/task"
)

const taskKeyReadFiles = "read-files"

// newTaskReadFiles creates a new task descriptor for the taskReadFiles task.
func newTaskReadFiles() TaskDescriptor {
	return TaskDescriptor{
		Key: taskKeyReadFiles,
	}
}

// taskReadFiles creates a task that reads files from a given directory.
// Only reads files with the generatedFilePrefix in their name.
func taskReadFiles(config Config) task.Task {
	return func(ctx context.Context, logger sharedlogging.Logger,
		scheduler task.Scheduler[TaskDescriptor]) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(10 * time.Second):
				dir, err := os.ReadDir(config.Directory)
				if err != nil {
					logger.Errorf("Error opening directory '%s': %s", config.Directory, err)

					continue
				}

				// iterate over all files in the directory.
				for _, file := range dir {
					// skip files that do not match the generatedFilePrefix.
					if !strings.HasPrefix(file.Name(), generatedFilePrefix) {
						continue
					}

					// schedule a task to ingest the file into the payments system.
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
