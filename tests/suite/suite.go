package suite

import (
	"context"
	"net"
	"strconv"
	"testing"

	"github.com/alexandernizov/grpcmessanger/api/gen/messanger/authpb"
	"github.com/alexandernizov/grpcmessanger/api/gen/messanger/chatpb"
	"github.com/alexandernizov/grpcmessanger/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	grpcHost = "localhost"
)

type Suite struct {
	*testing.T
	Cfg        *config.Config
	AuthClient authpb.AuthClient
	ChatClient chatpb.ChatClient
}

func New(t *testing.T) (context.Context, *Suite) {
	t.Helper()
	t.Parallel()

	cfg := config.MustLoadByPath("../configs/local.yaml")

	ctx, cancelCtx := context.WithTimeout(context.Background(), cfg.GrpcConfig.RequestTimeout)

	t.Cleanup(func() {
		t.Helper()
		cancelCtx()
	})

	cc, err := grpc.DialContext(
		context.Background(),
		grpcAddress(cfg),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc server connection failed: %v", err)
	}
	return ctx, &Suite{
		T:          t,
		Cfg:        cfg,
		AuthClient: authpb.NewAuthClient(cc),
		ChatClient: chatpb.NewChatClient(cc),
	}
}

func grpcAddress(cfg *config.Config) string {
	return net.JoinHostPort(grpcHost, strconv.Itoa(cfg.GrpcConfig.Port))
}
