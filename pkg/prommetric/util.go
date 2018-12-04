package prommetric

import (
	"strings"
)

const (
	noneNamespace = "______"
)

func joins(strs []string) string {
	sep := "|"
	strSize := len(strs)
	if strSize == 0 {
		return noneNamespace
	}

	lastIdx := strSize - 1
	builder := &strings.Builder{}
	for idx, str := range strs {
		builder.WriteString(str)
		if idx < lastIdx {
			builder.WriteString(sep)
		}
	}

	return builder.String()
}
