package utils

import (
	"encoding/json"
)

func JSON(v interface{}) string {
	jsons, err := json.Marshal(v)
	if err != nil {
		return `{"error":"` + err.Error() + `"}`
	}

	return string(jsons)
}

func JSONPretty(v interface{}) string {
	jsons, err := json.MarshalIndent(v, "\n", "\t")
	if err != nil {
		return `{"error":"` + err.Error() + `"}`
	}

	return string(jsons)
}
