package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
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

type MessagesProvder interface {
	GetNextOutboxMessage(ctx context.Context) (*domain.Message, error)
	ConfirmOutboxMessageSended(ctx context.Context, messageId int) error
}

type Publisher struct {
	log      *slog.Logger
	producer *kafka.Producer
	chats    ChatsProvider
	messages MessagesProvder
}

const (
	topicChats    = "chats"
	topicMessages = "messages"
	chatsTable    = "outbox_chats"
	messagesTable = "outbox_chats"
)

var (
	ErrNoConnection = errors.New("can't establish connection to kafka")
	ErrInternal     = errors.New("internal error")
)

type ConnectOptions struct {
	Host     string
	Port     string
	User     string
	Password string
	DBname   string
}

func New(log *slog.Logger, chatsProvider ChatsProvider, messagesProvider MessagesProvder, cOpts ConnectOptions) (*Publisher, error) {
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
	return &Publisher{log: log, producer: producer, chats: chatsProvider, messages: messagesProvider}, nil
}

func (p *Publisher) ServePublish() {
	go p.serveChats()
	go p.serveMessages()
}

func (p *Publisher) serveChats() {
	const op = "outbox.serveChats"
	log := p.log.With(slog.String("op", op))

	kafkaTopic := topicChats

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

		resChan := make(chan kafka.Event)

		err = p.producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &kafkaTopic, Partition: kafka.PartitionAny},
			Key:            []byte(chat.Uuid.String()),
			Value:          json,
		}, resChan)
		if err != nil {
			log.Info("error: ", sl.Err(err))
		}

		e := <-resChan
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				log.Warn("failed to deliver message", slog.Attr{Key: "topic", Value: slog.StringValue(ev.TopicPartition.String())})
			} else {
				if string(ev.Key) == chat.Uuid.String() {
					err = p.chats.ConfirmOutboxChatSended(context.TODO(), chat.Uuid)
					if err != nil {
						log.Warn("chat produced to kafka, but didn't match as delivered in postgres")
					}
				}
				log.Warn("produced event to topic", slog.Attr{Key: "topic", Value: slog.StringValue(ev.TopicPartition.String())},
					slog.Attr{Key: "key-valye", Value: slog.StringValue(string(ev.Key) + string(ev.Value))})
			}
		}
	}
}

func (p *Publisher) serveMessages() {
	const op = "outbox.serveMessages"
	log := p.log.With(slog.String("op", op))

	kafkaTopic := topicMessages

	for {
		time.Sleep(5 * time.Second)
		message, err := p.messages.GetNextOutboxMessage(context.TODO())
		if err != nil {
			if errors.Is(err, postgres.ErrNoOutboxMessages) {
				continue
			}
			log.Warn("error for getting next outbox message", sl.Err(err))
			continue
		}
		if message == nil {
			continue
		}

		json, err := json.Marshal(message)
		if err != nil {
			log.Warn("error for marshalling", sl.Err(err))
		}

		resChan := make(chan kafka.Event)

		err = p.producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &kafkaTopic, Partition: kafka.PartitionAny},
			Key:            []byte(strconv.Itoa(message.Id)),
			Value:          json,
		}, resChan)
		if err != nil {
			log.Info("error: ", sl.Err(err))
		}

		e := <-resChan
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				log.Warn("failed to deliver message", slog.Attr{Key: "topic", Value: slog.StringValue(ev.TopicPartition.String())})
			} else {
				if string(ev.Key) == strconv.Itoa(message.Id) {
					err = p.messages.ConfirmOutboxMessageSended(context.TODO(), message.Id)
					if err != nil {
						log.Warn("message produced to kafka, but didn't match as delivered in postgres")
					}
				}
				log.Warn("produced event to topic", slog.Attr{Key: "topic", Value: slog.StringValue(ev.TopicPartition.String())},
					slog.Attr{Key: "key-valye", Value: slog.StringValue(string(ev.Key) + string(ev.Value))})
			}
		}
	}
}
