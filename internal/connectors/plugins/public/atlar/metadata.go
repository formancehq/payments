package atlar

import (
	"fmt"
)

const (
	atlarMetadataSpecNamespace = "com.atlar.spec/"
	valueTRUE                  = "TRUE"
	valueFALSE                 = "FALSE"
)

func computeMetadata(key, value string) map[string]string {
	namespacedKey := fmt.Sprintf("%s%s", atlarMetadataSpecNamespace, key)
	return map[string]string{
		namespacedKey: value,
	}
}

func computeMetadataBool(key string, value bool) map[string]string {
	computedValue := valueFALSE
	if value {
		computedValue = valueTRUE
	}
	return computeMetadata(key, computedValue)
}

func extractNamespacedMetadata(metadata map[string]string, key string) (string, error) {
	value, ok := metadata[atlarMetadataSpecNamespace+key]
	if !ok {
		return "", fmt.Errorf("unable to find metadata with key %s%s", atlarMetadataSpecNamespace, key)
	}
	return value, nil
}
