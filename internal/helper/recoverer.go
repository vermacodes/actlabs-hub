package helper

import (
	"actlabs-hub/internal/logger"
	"context"
	"errors"
)

func Recoverer(ctx context.Context, maxRetries int, id string, f func()) {
	defer func() {
		if r := recover(); r != nil {
			var err error
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}
			logger.LogError(ctx, "recovering goroutine panic",
				"id", id,
				"err", err,
				"retry", maxRetries,
				"remaining retries", maxRetries-1,
			)
			if maxRetries == 0 {
				logger.LogError(ctx, "max retries exceeded, not retrying",
					"id", id,
				)
				panic("max retries exceeded, not retrying")
			} else {
				go Recoverer(ctx, maxRetries-1, id, f)
			}
		}
	}()
	logger.LogInfo(ctx, "starting goroutine",
		"id", id,
	)
	f() // call the function
}
