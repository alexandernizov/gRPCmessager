package grpc

import (
	"context"
	"errors"

	chatServ "github.com/alexandernizov/grpcmessanger/internal/services/chat"

	"github.com/alexandernizov/grpcmessanger/api/gen/chatpb"
	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/jwt"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChatProvider interface {
	NewChat(ctx context.Context, ownerUuid uuid.UUID, readonly bool, ttl int) (*domain.Chat, error)
	NewMessage(ctx context.Context, chatUuid uuid.UUID, authorUuid uuid.UUID, message string) (*domain.Message, error)
	ChatHistory(ctx context.Context, chatUuid uuid.UUID) ([]*domain.Message, error)
}

type ChatServer struct {
	chatpb.UnimplementedChatServer
	Provider ChatProvider
	Secret   string
}

func (c *ChatServer) NewChat(ctx context.Context, req *chatpb.NewChatReq) (*chatpb.NewChatResp, error) {
	if req.TtlSecs < 0 {
		return nil, status.Error(codes.InvalidArgument, "ttl should be more than 0")
	}

	ownerUuid, err := jwt.GetUserUuidFromToken(req.Token, []byte(c.Secret))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "token is incorrect")
	}

	chat, err := c.Provider.NewChat(ctx, ownerUuid, req.Readonly, int(req.TtlSecs))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	//Send response
	return &chatpb.NewChatResp{Uuid: chat.Uuid.String()}, nil
}

func (c *ChatServer) NewMessage(ctx context.Context, req *chatpb.NewMessageReq) (*chatpb.NewMessageResp, error) {
	if len(req.ChatUuid) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Chat UUID is required")
	}

	if len(req.Message) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Message is required")
	}

	authorUuid, err := jwt.GetUserUuidFromToken(req.Token, []byte(c.Secret))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Token is incorrect")
	}

	chatUuid, err := uuid.Parse(req.ChatUuid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Chat Uuid is incorrect")
	}

	_, err = c.Provider.NewMessage(ctx, chatUuid, authorUuid, req.Message)
	if err != nil {
		if errors.Is(err, chatServ.ErrChatNotFound) {
			return nil, status.Error(codes.NotFound, "Chat is not found")
		}
		if errors.Is(err, chatServ.ErrPermissionDenied) {
			return nil, status.Error(codes.PermissionDenied, "Chat is not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &chatpb.NewMessageResp{Published: true}, nil
}

func (c *ChatServer) ChatHistory(ctx context.Context, req *chatpb.ChatHistoryReq) (*chatpb.ChatHistoryResp, error) {
	if len(req.Uuid) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Chat UUID is required")
	}

	chatUuid, err := uuid.Parse(req.Uuid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Chat Uuid is incorrect")
	}

	res, err := c.Provider.ChatHistory(ctx, chatUuid)
	if err != nil {
		if errors.Is(err, chatServ.ErrChatNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	var messagesResponse []*chatpb.Message

	for _, message := range res {
		var mes chatpb.Message
		mes.Author = message.AuthorUuid.String()
		mes.Message = message.Body
		mes.Published = message.Published.Unix()
		messagesResponse = append(messagesResponse, &mes)
	}

	return &chatpb.ChatHistoryResp{Messages: messagesResponse}, nil
}
