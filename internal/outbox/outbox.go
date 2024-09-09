package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/logger/sl"
	"github.com/alexandernizov/grpcmessanger/internal/storage/postgres"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/google/uuid"
)

type ChatsProvider interface {
	GetNextOutboxChat(ctx context.Context) (*domain.Chat, error)
	ConfirmOutboxChatSended(ctx context.Context, chatUuid uuid.UUID) error
}

type Publisher struct {
	log      *slog.Logger
	producer *kafka.Producer
	chats    ChatsProvider
}

const (
	topicChat = "chats"
	chatTable = "outbox_chats"
)

var (
	ErrNoConnection  = errors.New("can't establish connection to kafka")
	ErrInternal      = errors.New("internal error")
	ErrUserNotFound  = errors.New("user is not found")
	ErrTokenNotFound = errors.New("token is not found")
)

type ConnectOptions struct {
	Host     string
	Port     string
	User     string
	Password string
	DBname   string
}

func New(log *slog.Logger, chatsProvider ChatsProvider, cOpts ConnectOptions) (*Publisher, error) {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":        cOpts.Host + ":" + cOpts.Port,
		"acks":                     "all",
		"socket.keepalive.enable":  true,
		"reconnect.backoff.ms":     100,
		"reconnect.backoff.max.ms": 10000,
	})
	if err != nil {
		return nil, fmt.Errorf("can't connect to Kafka: %w", ErrNoConnection)
	}
	return &Publisher{log: log, producer: producer, chats: chatsProvider}, nil
}

func (p *Publisher) ServePublish() {
	go func() {
		const op = "postgres.ServePublish"
		log := p.log.With(slog.String("op", op))

		for e := range p.producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Warn("failed to deliver message", slog.Attr{Key: "topic", Value: slog.StringValue(ev.TopicPartition.String())})
				} else {
					log.Warn("produced event to topic", slog.Attr{Key: "topic", Value: slog.StringValue(ev.TopicPartition.String())},
						slog.Attr{Key: "key-valye", Value: slog.StringValue(string(ev.Key) + string(ev.Value))})
				}
			}
		}
	}()

	go func() {
		const op = "postgres.ServePublish"
		log := p.log.With(slog.String("op", op))

		kafkaTopic := topicChat

		for {
			time.Sleep(5 * time.Second)
			chat, err := p.chats.GetNextOutboxChat(context.TODO())
			if err != nil {
				if errors.Is(err, postgres.ErrNoOutboxChats) {
					continue
				}
				log.Warn("error for getting next outbox chat", sl.Err(err))
				continue
			}
			if chat == nil {
				continue
			}

			json, err := json.Marshal(chat)
			if err != nil {
				log.Warn("error for marshalling", sl.Err(err))
			}

			err = p.producer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &kafkaTopic, Partition: kafka.PartitionAny},
				Key:            []byte(chat.Uuid.String()),
				Value:          json,
			}, nil)
			if err != nil {
				log.Info("error: ", sl.Err(err))
			}

			p.producer.Flush(15 * 1000)
		}
	}()
}
