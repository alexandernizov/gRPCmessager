package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/IBM/sarama"
	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/logger/sl"
	"github.com/alexandernizov/grpcmessanger/internal/storage"
	"github.com/google/uuid"
)

type OutboxProvider interface {
	GetNextOutbox(ctx context.Context) (*domain.Outbox, error)
	ConfirmOutboxSended(ctx context.Context, outboxUuid uuid.UUID) error
}

type Publisher struct {
	log      *slog.Logger
	producer sarama.SyncProducer
	outbox   OutboxProvider
	stopChan chan struct{}
}

var (
	ErrNoConnection = errors.New("can't establish connection to kafka")
	ErrInternal     = errors.New("internal error")
)

func New(log *slog.Logger, outboxProvider OutboxProvider, brokers []string) (*Publisher, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("can't start sarama producer: %w", ErrNoConnection)
	}
	return &Publisher{log: log, producer: producer, outbox: outboxProvider}, nil
}

func (p *Publisher) Start() {
	const op = "publisher.ProduceMessage"
	log := p.log.With(slog.String("op", op))

	ctx := context.Background()
	go func() {
		for {
			select {
			case <-p.stopChan:
				p.producer.Close()
				return
			default:
				nextOutbox, err := p.outbox.GetNextOutbox(ctx)
				if err != nil {
					if errors.Is(err, storage.ErrNoOutbox) {
						continue
					}
					log.Error("error with getting next outbox message", sl.Err(err))
					//TODO: write it to deadqueue
				}
				if nextOutbox != nil {
					msg := &sarama.ProducerMessage{
						Topic: nextOutbox.Topic,
						Key:   sarama.StringEncoder(nextOutbox.Uuid.String()),
						Value: sarama.ByteEncoder(nextOutbox.Message),
					}
					_, _, err := p.producer.SendMessage(msg)
					if err != nil {
						//TODO: write it to deadqueue
						log.Error("error with producing outbox message", sl.Err(err))
					}
					err = p.outbox.ConfirmOutboxSended(ctx, nextOutbox.Uuid)
					if err != nil {
						log.Error("error to confirm message", sl.Err(err))
					}
				}
			}
		}
	}()
}

func (p *Publisher) Stop() {
	p.stopChan <- struct{}{}
}
