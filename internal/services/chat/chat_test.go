package chat

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/services/chat/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

var (
	ownerUuidTest = uuid.MustParse("8ee4e645-b894-4477-820b-48381e10677f")
	ownerTest     = domain.User{Uuid: ownerUuidTest, Login: "test", PasswordHash: []byte("test")}
	chatUuidTest  = uuid.MustParse("30d88aa9-b8a5-4cfb-af4b-c043278e111e")
	deadlineTest  = time.Now()
	publishedTest = time.Now()
)

type mockArgs struct {
	methodName string
	arguments  []any
	returning  []any
}

func NewMockService(t *testing.T, inputMocks []mockArgs) *ChatService {
	chatStorage := mocks.NewChatStorage(t)
	for _, m := range inputMocks {
		chatStorage.On(m.methodName, m.arguments...).Return(m.returning...).Once()
	}
	mockService := ChatService{
		chatStorage: chatStorage,
		chatOptions: ChatOptions{MaximumCount: 1, MaximumMessages: 1},
	}
	return &mockService
}

func TestChatService_NewChat(t *testing.T) {
	type funcArgs struct {
		ownerUuid uuid.UUID
		readonly  bool
		ttl       int
	}
	tests := []struct {
		name     string
		funcArgs funcArgs
		mockArgs []mockArgs
		want     *domain.Chat
		wantErr  bool
	}{
		{
			name: "success",
			funcArgs: funcArgs{
				ownerUuid: ownerUuidTest,
				readonly:  false,
				ttl:       10,
			},
			mockArgs: []mockArgs{
				{methodName: "ChatsCount", arguments: []any{mock.Anything}, returning: []any{0, nil}},
				{methodName: "CreateChat", arguments: []any{mock.Anything, mock.Anything}, returning: []any{&domain.Chat{Uuid: chatUuidTest, Owner: ownerTest, Readonly: false, Deadline: deadlineTest}, nil}},
			},
			want:    &domain.Chat{Uuid: chatUuidTest, Owner: ownerTest, Readonly: false, Deadline: deadlineTest},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewMockService(t, tt.mockArgs)
			got, err := c.NewChat(context.TODO(), tt.funcArgs.ownerUuid, tt.funcArgs.readonly, tt.funcArgs.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChatService.NewChat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ChatService.NewChat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChatService_NewMessage(t *testing.T) {
	type funcArgs struct {
		ctx        context.Context
		chatUuid   uuid.UUID
		authorUuid uuid.UUID
		message    string
	}
	tests := []struct {
		name     string
		funcArgs funcArgs
		mockArgs []mockArgs
		want     *domain.Message
		wantErr  bool
	}{
		{
			name: "success",
			funcArgs: funcArgs{
				ctx:        context.TODO(),
				chatUuid:   chatUuidTest,
				authorUuid: ownerUuidTest,
				message:    "test",
			},
			mockArgs: []mockArgs{
				{methodName: "GetChat", arguments: []any{mock.Anything, chatUuidTest}, returning: []any{&domain.Chat{Uuid: chatUuidTest, Owner: ownerTest, Readonly: false, Deadline: deadlineTest}, nil}},
				{methodName: "PostMessage", arguments: []any{mock.Anything, chatUuidTest, mock.Anything}, returning: []any{&domain.Message{Id: 1, AuthorUuid: ownerUuidTest, Body: "test", Published: publishedTest}, nil}},
				{methodName: "TrimMessages", arguments: []any{mock.Anything, chatUuidTest, 1}, returning: []any{true, nil}},
			},
			want:    &domain.Message{Id: 1, AuthorUuid: ownerUuidTest, Body: "test", Published: publishedTest},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewMockService(t, tt.mockArgs)
			got, err := c.NewMessage(tt.funcArgs.ctx, tt.funcArgs.chatUuid, tt.funcArgs.authorUuid, tt.funcArgs.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChatService.NewMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ChatService.NewMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChatService_ChatHistory(t *testing.T) {
	type funcArgs struct {
		ctx      context.Context
		chatUuid uuid.UUID
	}
	tests := []struct {
		name     string
		funcArgs funcArgs
		mockArgs []mockArgs
		want     []*domain.Message
		wantErr  bool
	}{
		{
			name: "success",
			funcArgs: funcArgs{
				ctx:      context.TODO(),
				chatUuid: chatUuidTest,
			},
			mockArgs: []mockArgs{
				{methodName: "GetChatHistory", arguments: []any{mock.Anything, chatUuidTest}, returning: []any{[]*domain.Message{}, nil}},
			},
			want:    []*domain.Message{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewMockService(t, tt.mockArgs)
			got, err := c.ChatHistory(tt.funcArgs.ctx, tt.funcArgs.chatUuid)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChatService.ChatHistory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ChatService.ChatHistory() = %v, want %v", got, tt.want)
			}
		})
	}
}
