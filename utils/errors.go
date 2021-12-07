package utils

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

func LogErrorf(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	log.Error(err)
	return err
}
