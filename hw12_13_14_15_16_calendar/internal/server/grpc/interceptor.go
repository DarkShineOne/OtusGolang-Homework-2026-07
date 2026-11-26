package internalgrpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func loggingInterceptor(logger Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		clientAddr := "-"
		if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
			clientAddr = p.Addr.String()
		}
		logger.Info(fmt.Sprintf(
			`%s [%s] %s %s %s`,
			clientAddr,
			start.Format("02/Jan/2006:15:04:05 -0700"),
			info.FullMethod,
			status.Code(err),
			time.Since(start),
		))
		return resp, err
	}
}
