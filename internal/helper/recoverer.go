package helper

import (
	"errors"

	"golang.org/x/exp/slog"
)

func Recoverer(maxRetries int, id string, f func()) {
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
			slog.Error("recovering goroutine panic",
				slog.String("id", id),
				slog.String("err", err.Error()),
				slog.Int("retry", maxRetries),
				slog.Int("remaining retries", maxRetries-1),
			)
			if maxRetries == 0 {
				slog.Error("max retries exceeded, not retrying",
					slog.String("id", id),
				)
				panic("max retries exceeded, not retrying")
			} else {
				go Recoverer(maxRetries-1, id, f)
			}
		}
	}()
	slog.Info("starting goroutine",
		slog.String("id", id),
	)
	f() // call the function
}
