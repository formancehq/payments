package engine

import (
	"fmt"
)

func GetDefaultTaskQueue(stack string) string {
	return fmt.Sprintf("%s-default", stack)
}
