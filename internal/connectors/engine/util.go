package engine

import (
	"fmt"
)

func getDefaultTaskQueue(stack string) string {
	return fmt.Sprintf("%s-default", stack)
}
