package util

import (
	"go.uber.org/zap"
)

var Sugar *zap.SugaredLogger

func init() {
	logger, err := zap.NewProduction()
	if err != nil {
		Sugar.Infof(err.Error())
	}

	Sugar = logger.Sugar()
}

func Load() {
	logger, err := zap.NewProduction()
	if err != nil {
		Sugar.Infof("can't initialize zap logger: %v", err)
	}

	defer func() {
		err = logger.Sync()

		if err != nil {
			Sugar.Infof("can't Sync zap logger: %v", err)
		}
	}()
}
