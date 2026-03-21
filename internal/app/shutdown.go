package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const gracefulShutdownTimeout = 5 * time.Second

// WaitForShutdown blocks until SIGINT or SIGTERM is received, then calls
// svc.Shutdown with a 5-second graceful timeout.
func WaitForShutdown(svc *ConnectionService) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	ctx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()

	svc.Shutdown(ctx)
}
