package utils

import (
	"encoding/json"
	"os"
)

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ParseRawJSON(data string) (map[string]interface{}, error) {
	parsedData := make(map[string]interface{})
	err := json.Unmarshal([]byte(data), &parsedData)

	if err != nil {
		return nil, err
	}

	return parsedData, nil
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
