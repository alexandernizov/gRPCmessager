package grpc

import (
	"context"

	"github.com/alexandernizov/grpcmessanger/api/gen/chatpb"
	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/google/uuid"
)

type ChatProvider interface {
	NewChat(ctx context.Context, ownerUuid uuid.UUID, readonly bool, ttl int) (uuid.UUID, error)
	NewMessage(ctx context.Context, authorUuid uuid.UUID, chatUuid uuid.UUID, message string) (bool, error)
	ChatHistory(ctx context.Context, chatUuid uuid.UUID) ([]domain.Message, error)
}

type ChatServer struct {
	chatpb.UnimplementedChatServer
	Provider ChatProvider
	Secret   string
}

func (c *ChatServer) NewChat(ctx context.Context, req *chatpb.NewChatReq) (*chatpb.NewChatResp, error) {
	// //Validate
	// if len(req.Token) == 0 {
	// 	return nil, status.Error(codes.InvalidArgument, "Token is required")
	// }

	// if req.TtlSecs < 0 {
	// 	return nil, status.Error(codes.InvalidArgument, "TTL should be more than 0")
	// }

	// ownerUuid, err := jwt.GetUserUuidFromToken(req.Token, []byte(c.Secret))
	// if err != nil {
	// 	return nil, status.Error(codes.InvalidArgument, "Token is incorrect")
	// }

	// chatUuid, err := c.Provider.NewChat(ctx, ownerUuid, req.Readonly, int(req.TtlSecs))
	// if err != nil {
	// 	return nil, status.Error(codes.Internal, err.Error())
	// }
	// //Send response
	// return &chatpb.NewChatResp{Uuid: chatUuid.String()}, nil

	return nil, nil
}

func (c *ChatServer) NewMessage(ctx context.Context, req *chatpb.NewMessageReq) (*chatpb.NewMessageResp, error) {
	// if len(req.Token) == 0 {
	// 	return nil, status.Error(codes.InvalidArgument, "Token is required")
	// }

	// if len(req.ChatUuid) == 0 {
	// 	return nil, status.Error(codes.InvalidArgument, "Chat UUID is required")
	// }

	// if len(req.Message) == 0 {
	// 	return nil, status.Error(codes.InvalidArgument, "Message is required")
	// }

	// authorUuid, err := jwt.GetUserUuidFromToken(req.Token, []byte(c.Secret))
	// if err != nil {
	// 	return nil, status.Error(codes.InvalidArgument, "Token is incorrect")
	// }

	// chatUuid, err := uuid.Parse(req.ChatUuid)
	// if err != nil {
	// 	return nil, status.Error(codes.InvalidArgument, "Chat Uuid is incorrect")
	// }

	// res, err := c.Provider.NewMessage(ctx, authorUuid, chatUuid, req.Message)
	// if err != nil {
	// 	return nil, status.Error(codes.Internal, err.Error())
	// }

	// return &chatpb.NewMessageResp{Published: res}, nil

	return nil, nil
}

func (c *ChatServer) ChatHistory(ctx context.Context, req *chatpb.ChatHistoryReq) (*chatpb.ChatHistoryResp, error) {
	// if len(req.Token) == 0 {
	// 	return nil, status.Error(codes.InvalidArgument, "Token is required")
	// }

	// if len(req.Uuid) == 0 {
	// 	return nil, status.Error(codes.InvalidArgument, "Chat UUID is required")
	// }

	// ok, err := jwt.ValidateToken(req.Token, []byte(c.Secret))
	// if !ok || err != nil {
	// 	return nil, status.Error(codes.InvalidArgument, "Token is incorrect")
	// }

	// chatUuid, err := uuid.Parse(req.Uuid)
	// if err != nil {
	// 	return nil, status.Error(codes.InvalidArgument, "Chat Uuid is incorrect")
	// }

	// res, err := c.Provider.ChatHistory(ctx, chatUuid)
	// if err != nil {
	// 	return nil, status.Error(codes.Internal, err.Error())
	// }

	// var messagesResponse []*chatpb.Message

	// for _, message := range res {
	// 	var mes chatpb.Message
	// 	mes.Uuid = message.Uuid
	// 	mes.Author = message.AuthorUuid
	// 	mes.Message = message.Body
	// 	mes.Published = message.Published
	// 	messagesResponse = append(messagesResponse, &mes)
	// }

	// return &chatpb.ChatHistoryResp{Messages: messagesResponse}, nil

	return nil, nil
}
