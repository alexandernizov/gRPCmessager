package inmemory

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/domain/errs"
	"github.com/google/uuid"
)

type ChatMemStorage struct {
	MaxChatsCount     int
	CurrentChatsCount int
	MaxMessagesInChat int
	mux               sync.RWMutex
	Chats             []domain.Chat
}

func NewChatMemStorage(maxChatsCount int, maxMessagesInChat int) *ChatMemStorage {
	chatsSlice := make([]domain.Chat, maxChatsCount)
	chatStorage := ChatMemStorage{MaxChatsCount: maxChatsCount, MaxMessagesInChat: maxMessagesInChat, Chats: chatsSlice}
	return &chatStorage
}

func (c *ChatMemStorage) MakeNewChat(ctx context.Context, owner *domain.User, readonly bool, ttl int) (uuid.UUID, error) {
	//
	newChat := domain.Chat{
		Uuid:     uuid.New(),
		Owner:    *owner,
		ReadOnly: readonly,
		Capacity: c.MaxMessagesInChat,
		Mux:      &sync.RWMutex{},
		Messages: make(map[uuid.UUID]*domain.Message),
		Queue:    list.New(),
	}

	c.Chats = append(c.Chats, newChat)
	c.autoDeleteChat(newChat.Uuid, ttl)

	return newChat.Uuid, nil
}

func (c *ChatMemStorage) autoDeleteChat(chat uuid.UUID, ttl int) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(ttl))

	go func() {
		<-ctx.Done()
		defer cancel()

		c.mux.Lock()
		defer c.mux.Unlock()

		chatIdx := -1
		for i, v := range c.Chats {
			if v.Uuid == chat {
				chatIdx = i
			}
		}
		if chatIdx == -1 {
			return
		}

		newChatsSlice := make([]domain.Chat, c.MaxChatsCount)
		newChatsSlice = append(newChatsSlice, c.Chats[:chatIdx]...)
		newChatsSlice = append(newChatsSlice, c.Chats[chatIdx+1:]...)

		c.Chats = newChatsSlice
	}()
}

func (c *ChatMemStorage) PostMessage(ctx context.Context, author *domain.User, chat uuid.UUID, message string) (bool, error) {
	// Find chat
	chatForMessage, err := c.getChat(chat)
	if err != nil {
		return false, err
	}

	// Check Capacity
	if chatForMessage.Queue.Len() >= c.MaxMessagesInChat {
		c.purgeChat(chatForMessage)
	}

	// Make message
	newMessage := domain.Message{Uuid: uuid.New(), Author: *author, Published: time.Now(), Message: message}
	chatForMessage.Mux.Lock()
	defer chatForMessage.Mux.Unlock()

	// Add message uuid to queue
	chatForMessage.Queue.PushFront(newMessage.Uuid)

	// Add message
	chatForMessage.Messages[newMessage.Uuid] = &newMessage

	return true, nil
}

func (c *ChatMemStorage) GetChatHistory(ctx context.Context, chat uuid.UUID) ([]domain.Message, error) {
	var res []domain.Message
	findedChat, err := c.getChat(chat)

	if err != nil {
		return res, err
	}

	findedChat.Mux.RLock()
	defer findedChat.Mux.RUnlock()

	for _, message := range findedChat.Messages {
		res = append(res, *message)
	}
	return res, nil
}

func (c *ChatMemStorage) getChat(chat uuid.UUID) (*domain.Chat, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	for _, v := range c.Chats {
		if v.Uuid == chat {
			return &v, nil
		}
	}
	return nil, errs.ErrChatDoesNotExist
}

func (c *ChatMemStorage) purgeChat(chat *domain.Chat) {
	chat.Mux.Lock()
	defer chat.Mux.Unlock()

	if message := chat.Queue.Back(); message != nil {
		item := chat.Queue.Remove(message).(uuid.UUID)
		delete(chat.Messages, item)
	}
}

func (c *ChatMemStorage) IsChatReadOnly(ctx context.Context, chatUuid uuid.UUID) (bool, *domain.User, error) {
	chat, err := c.getChat(chatUuid)
	if err != nil {
		return false, nil, err
	}
	return chat.ReadOnly, &chat.Owner, nil
}

func (c *ChatMemStorage) GetMessage(ctx context.Context, message uuid.UUID) (*domain.Message, error) {
	c.mux.Lock()
	defer c.mux.Unlock()
	for _, chat := range c.Chats {
		msg, exists := chat.Messages[message]
		if !exists {
			continue
		}
		for e := chat.Queue.Front(); e != nil; e = e.Next() {
			if e.Value == msg.Uuid {
				chat.Queue.Remove(e)
				break
			}
		}
		chat.Queue.PushFront(msg.Uuid)
		return msg, nil
	}
	return nil, errs.ErrMessageNotFound
}

func (c *ChatMemStorage) UpdateMessage(ctx context.Context, message *domain.Message, newText string) (bool, error) {
	message.Message = newText
	return true, nil
}
