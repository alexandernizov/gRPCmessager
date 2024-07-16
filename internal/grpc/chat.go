package grpc

import (
	"context"
	"errors"

	"github.com/alexandernizov/grpcmessanger/api/gen/messanger/chatpb"
	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/domain/errs"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChatProvider interface {
	NewChat(ctx context.Context, owner *domain.User, readonly bool, ttl int) (uuid.UUID, error)
	NewMessage(ctx context.Context, author *domain.User, chatUuid uuid.UUID, message string) (bool, error)
	EditMessage(ctx context.Context, author *domain.User, messageUuid uuid.UUID, message string) (bool, error)
	ChatHistory(ctx context.Context, chat uuid.UUID) ([]domain.Message, error)
}

type chatServer struct {
	chatpb.UnimplementedChatServer
	provider ChatProvider
}

func (c *chatServer) NewChat(ctx context.Context, req *chatpb.NewChatReq) (*chatpb.NewChatResp, error) {
	// Проверяем параметры
	user, err := getUserFromCtx(ctx)
	if user == nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	if req.TtlSecs < 0 {
		return nil, status.Error(codes.InvalidArgument, "TTL should be more than 0")
	}

	chatUuid, err := c.provider.NewChat(ctx, user, (req.Readonly == chatpb.ReadOnly_TRUE), int(req.TtlSecs))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &chatpb.NewChatResp{Uuid: chatUuid.String()}, nil
}

func (c *chatServer) NewMessage(ctx context.Context, req *chatpb.NewMessageReq) (*chatpb.NewMessageResp, error) {
	user, err := getUserFromCtx(ctx)
	if user == nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	chatUuid, err := uuid.Parse(req.ChatUuid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if len(req.Message) <= 0 {
		return nil, status.Error(codes.InvalidArgument, "message shouldn't be empty")
	}

	res, err := c.provider.NewMessage(ctx, user, chatUuid, req.Message)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &chatpb.NewMessageResp{Published: res}, nil

}
func (c *chatServer) ChatHistory(ctx context.Context, req *chatpb.ChatHistoryReq) (*chatpb.ChatHistoryResp, error) {
	chatUuid, err := uuid.Parse(req.Uuid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	messages, err := c.provider.ChatHistory(ctx, chatUuid)
	if errors.Is(err, errs.ErrChatDoesNotExist) {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var res []*chatpb.Message
	for _, m := range messages {
		res = append(res, &chatpb.Message{
			Uuid:      m.Uuid.String(),
			Author:    m.Author.Name,
			Published: m.Published.Unix(),
			Message:   m.Message,
		})
	}

	return &chatpb.ChatHistoryResp{Messages: res}, nil
}

func (c *chatServer) EditMessage(ctx context.Context, req *chatpb.EditMessageReq) (*chatpb.EditMessageResp, error) {
	user, err := getUserFromCtx(ctx)
	if user == nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	messageUuid, err := uuid.Parse(req.Uuid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if len(req.Message) <= 0 {
		return nil, status.Error(codes.InvalidArgument, "message shouldn't be empty")
	}

	res, err := c.provider.EditMessage(ctx, user, messageUuid, req.Message)
	if errors.Is(err, errs.ErrMessageNotFound) {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &chatpb.EditMessageResp{Published: res}, nil
}

func getUserFromCtx(ctx context.Context) (*domain.User, error) {
	user := ctx.Value(domain.UserCtxKey{}).(*domain.User)
	if user == nil {
		return nil, errs.ErrUserNotFound
	}
	return user, errs.ErrUserNotFound
}
