package dummypay

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/afero"

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
func taskReadFiles(config Config, fs fs) task.Task {
	return func(ctx context.Context, logger sharedlogging.Logger,
		scheduler task.Scheduler[TaskDescriptor]) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(config.FilePollingPeriod):
				files, err := parseFilesToIngest(config, fs)
				if err != nil {
					return fmt.Errorf("error parsing files to ingest: %w", err)
				}

				for _, file := range files {
					// schedule a task to ingest the file into the payments system.
					err = scheduler.Schedule(newTaskIngest(file), true)
					if err != nil {
						return fmt.Errorf("failed to schedule task to ingest file '%s': %w", file, err)
					}
				}
			}
		}
	}
}

func parseFilesToIngest(config Config, fs fs) ([]string, error) {
	dir, err := afero.ReadDir(fs, config.Directory)
	if err != nil {
		return nil, fmt.Errorf("error reading directory '%s': %w", config.Directory, err)
	}

	var files []string

	// iterate over all files in the directory.
	for _, file := range dir {
		// skip files that do not match the generatedFilePrefix.
		if !strings.HasPrefix(file.Name(), generatedFilePrefix) {
			continue
		}

		files = append(files, file.Name())

	}

	return files, nil
}
