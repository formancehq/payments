package gocardless

import "fmt"

func extractNamespacedMetadata(metadata map[string]string, key string) (string, error) {
	value, ok := metadata[key]
	if !ok {
		return "", fmt.Errorf("unable to find metadata with key %s", key)
	}
	return value, nil
}
