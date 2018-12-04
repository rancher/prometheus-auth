package utils

import (
	"fmt"
	log "github.com/sirupsen/logrus"
)

type tracker struct {
}

func (t *tracker) LogTrace(msg interface{}) {
	if log.GetLevel() != log.DebugLevel {
		return
	}

	switch m := msg.(type) {
	case func() string:
		fmt.Println(m())
	default:
		fmt.Println(m)
	}
}

var (
	stdTracker = &tracker{}
)

func LogTrace(msg interface{}) {
	stdTracker.LogTrace(msg)
}
