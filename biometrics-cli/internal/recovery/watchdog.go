package recovery

import (
	"context"
	"log/slog"
	"time"
)

// Watchdog überwacht einen Context und bricht ihn nach einem Timeout ab
func StartWatchdog(logger *slog.Logger, taskID string, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			logger.Error("WATCHDOG TRIGGERED: Task exceeded timeout, killing process group!",
				slog.String("task_id", taskID),
				slog.String("timeout", timeout.String()),
			)
		}
	}()

	return ctx, cancel
}
