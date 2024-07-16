package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func unaryLoggingInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

		log.Info(fmt.Sprintf("Request: %s", info.FullMethod))

		resp, err := handler(ctx, req)

		if err != nil {
			st := status.Convert(err)
			reqJSON, _ := json.Marshal(req)

			switch st.Code() {
			case codes.Unauthenticated:
				log.Warn(fmt.Sprintf("Unauthenticated try: %s Request: %s", info.FullMethod, reqJSON))
			default:
				log.Warn(fmt.Sprintf("Request error: %s, %s", st.Code().String(), reqJSON))
			}
		}

		return resp, err
	}
}
