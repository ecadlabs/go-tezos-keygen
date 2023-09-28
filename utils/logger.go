package utils

import (
	log "github.com/sirupsen/logrus"
)

type DebugLogger log.Logger

func (l *DebugLogger) Printf(format string, a ...any) {
	(*log.Logger)(l).Debugf(format, a...)
}
