package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/broady/tygor"
)

// LoggingInterceptor creates an interceptor that logs RPC calls using slog.
// It logs the start and end of each RPC call, including duration and error status.
func LoggingInterceptor(logger *slog.Logger) tygor.UnaryInterceptor {
	if logger == nil {
		logger = slog.Default()
	}

	return func(ctx context.Context, req any, info *tygor.RPCInfo, handler tygor.HandlerFunc) (any, error) {
		start := time.Now()

		logger.InfoContext(ctx, "RPC started",
			slog.String("service", info.Service),
			slog.String("method", info.Method),
		)

		res, err := handler(ctx, req)
		duration := time.Since(start)

		if err != nil {
			logger.ErrorContext(ctx, "RPC failed",
				slog.String("service", info.Service),
				slog.String("method", info.Method),
				slog.Duration("duration", duration),
				slog.Any("error", err),
			)
		} else {
			logger.InfoContext(ctx, "RPC completed",
				slog.String("service", info.Service),
				slog.String("method", info.Method),
				slog.Duration("duration", duration),
			)
		}

		return res, err
	}
}
