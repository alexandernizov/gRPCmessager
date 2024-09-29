package grpc

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/alexandernizov/grpcmessanger/api/gen/chatpb"
	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/grpc/mocks"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/jwt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

var (
	chatUuidForTests  = uuid.MustParse("01426c84-7593-46b1-8a10-a9bbe4a8bd4c")
	userUuidForTests  = uuid.MustParse("4c92f03d-cdbe-40c5-8edd-3a938aa69e50")
	userForTests      = domain.User{Uuid: userUuidForTests, Login: "Test", PasswordHash: []byte("Test")}
	secretTests       = []byte("Test")
	tokensForTests, _ = jwt.NewTokens(userForTests, 10000, 10000, secretTests)
	publishedForTest  = time.Now()
)

func TestChatServer_NewChat(t *testing.T) {
	type mockArgs struct {
		methodName string
		arguments  []any
		returning  []any
	}
	type funcArgs struct {
		ctx context.Context
		req *chatpb.NewChatReq
	}
	tests := []struct {
		name     string
		funcArgs funcArgs
		mockArgs mockArgs
		want     *chatpb.NewChatResp
		wantErr  bool
	}{
		{
			name: "success",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &chatpb.NewChatReq{
					Token:    tokensForTests.AccessToken,
					Readonly: false,
					TtlSecs:  1,
				},
			},
			mockArgs: mockArgs{methodName: "NewChat", arguments: []any{mock.Anything, userUuidForTests, false, 1}, returning: []any{&domain.Chat{Uuid: chatUuidForTests}, nil}},
			want:     &chatpb.NewChatResp{Uuid: chatUuidForTests.String()},
			wantErr:  false,
		},
		{
			name: "ttl_less_0",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &chatpb.NewChatReq{
					Token:    tokensForTests.AccessToken,
					Readonly: false,
					TtlSecs:  -1,
				},
			},
			mockArgs: mockArgs{},
			want:     nil,
			wantErr:  true,
		},
		{
			name: "incorrect_uuid",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &chatpb.NewChatReq{
					Token:    "incorrect uuid",
					Readonly: false,
					TtlSecs:  1,
				},
			},
			mockArgs: mockArgs{},
			want:     nil,
			wantErr:  true,
		},
		{
			name: "incorrect_uuid",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &chatpb.NewChatReq{
					Token:    tokensForTests.AccessToken,
					Readonly: false,
					TtlSecs:  1,
				},
			},
			mockArgs: mockArgs{methodName: "NewChat", arguments: []any{mock.Anything, userUuidForTests, false, 1}, returning: []any{nil, errors.New("some error")}},
			want:     nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatProvider := mocks.NewChatProvider(t)
			if tt.mockArgs.methodName > "" {
				chatProvider.On(tt.mockArgs.methodName, tt.mockArgs.arguments...).Return(tt.mockArgs.returning...).Once()
			}
			c := &ChatServer{
				Provider: chatProvider,
				Secret:   string(secretTests),
			}
			got, err := c.NewChat(tt.funcArgs.ctx, tt.funcArgs.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChatServer.NewChat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ChatServer.NewChat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChatServer_NewMessage(t *testing.T) {
	type mockArgs struct {
		methodName string
		arguments  []any
		returning  []any
	}
	type funcArgs struct {
		ctx context.Context
		req *chatpb.NewMessageReq
	}
	tests := []struct {
		name     string
		funcArgs funcArgs
		mockArgs mockArgs
		want     *chatpb.NewMessageResp
		wantErr  bool
	}{
		{
			name: "success",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &chatpb.NewMessageReq{
					Token:    tokensForTests.AccessToken,
					ChatUuid: chatUuidForTests.String(),
					Message:  "Test",
				},
			},
			mockArgs: mockArgs{methodName: "NewMessage", arguments: []any{mock.Anything, chatUuidForTests, userUuidForTests, "Test"}, returning: []any{nil, nil}},
			want:     &chatpb.NewMessageResp{Published: true},
			wantErr:  false,
		},
		{
			name: "empty_chat_uuid",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &chatpb.NewMessageReq{
					Token:   tokensForTests.AccessToken,
					Message: "Test",
				},
			},
			mockArgs: mockArgs{},
			want:     nil,
			wantErr:  true,
		},
		{
			name: "empty_message",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &chatpb.NewMessageReq{
					Token:    tokensForTests.AccessToken,
					ChatUuid: chatUuidForTests.String(),
					Message:  "",
				},
			},
			mockArgs: mockArgs{},
			want:     nil,
			wantErr:  true,
		},
		{
			name: "incorrect_token",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &chatpb.NewMessageReq{
					Token:    "incorrect token",
					ChatUuid: chatUuidForTests.String(),
					Message:  "Test",
				},
			},
			mockArgs: mockArgs{},
			want:     nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatProvider := mocks.NewChatProvider(t)
			if tt.mockArgs.methodName > "" {
				chatProvider.On(tt.mockArgs.methodName, tt.mockArgs.arguments...).Return(tt.mockArgs.returning...).Once()
			}
			c := &ChatServer{
				Provider: chatProvider,
				Secret:   string(secretTests),
			}
			got, err := c.NewMessage(tt.funcArgs.ctx, tt.funcArgs.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChatServer.NewMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ChatServer.NewMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChatServer_ChatHistory(t *testing.T) {
	type mockArgs struct {
		methodName string
		arguments  []any
		returning  []any
	}
	type funcArgs struct {
		ctx context.Context
		req *chatpb.ChatHistoryReq
	}
	tests := []struct {
		name     string
		funcArgs funcArgs
		mockArgs mockArgs
		want     *chatpb.ChatHistoryResp
		wantErr  bool
	}{
		{
			name: "success",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &chatpb.ChatHistoryReq{
					Token: userUuidForTests.String(),
					Uuid:  chatUuidForTests.String(),
				},
			},
			mockArgs: mockArgs{methodName: "ChatHistory", arguments: []any{mock.Anything, chatUuidForTests}, returning: []any{[]*domain.Message{
				{Id: 1, AuthorUuid: userUuidForTests, Body: "test", Published: publishedForTest},
			}, nil}},
			want: &chatpb.ChatHistoryResp{Messages: []*chatpb.Message{
				{Author: userUuidForTests.String(), Published: publishedForTest.Unix(), Message: "test"},
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatProvider := mocks.NewChatProvider(t)
			if tt.mockArgs.methodName > "" {
				chatProvider.On(tt.mockArgs.methodName, tt.mockArgs.arguments...).Return(tt.mockArgs.returning...).Once()
			}
			c := &ChatServer{
				Provider: chatProvider,
				Secret:   string(secretTests),
			}
			got, err := c.ChatHistory(tt.funcArgs.ctx, tt.funcArgs.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChatServer.ChatHistory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ChatServer.ChatHistory() = %v, want %v", got, tt.want)
			}
		})
	}
}
