package kafkaSender

import (
	"fmt"
	"github.com/Dmitrij-bot/marketserv/internal/repository"
	"github.com/IBM/sarama"
	"log"
	"time"
)

type Sender struct {
	r      repository.Interface
	stopCh chan struct{}
}

func NewSender(r repository.Interface) *Sender {
	return &Sender{
		r:      r,
		stopCh: make(chan struct{}),
	}
}

func (s *Sender) Start(handlePeriod time.Duration) {

	ticker := time.NewTicker(handlePeriod)

	go func() {
		for {
			select {
			case <-s.stopCh:
				log.Println("stopping event processing")
				return
			case <-ticker.C:
			}

			event, err := s.r.GetKafkaMessage()
			if err != nil {
				log.Printf("failed to get new event: %v", err)
				continue
			}
			if event.ID == 0 {
				log.Println("no new events")
				continue
			}

			kafkaEvent := Event{
				Key:     event.Key,
				Message: event.Message,
			}

			if err := s.sendKafkaMessage(kafkaEvent); err != nil {
				log.Printf("failed to send Kafka message: %v", err)
				continue
			}

			if err := s.r.SetDone(event.ID); err != nil {
				log.Printf("failed to set event done: %v", err)
				continue
			}
		}
	}()

}

func (s *Sender) Stop() {

	close(s.stopCh)
}

func (s *Sender) sendKafkaMessage(event Event) error {

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{"localhost:29092"}, config)
	if err != nil {
		return fmt.Errorf("ошибка создания Kafka producer: %w", err)
	}
	defer producer.Close()

	msg := &sarama.ProducerMessage{
		Topic: "test1",
		Key:   sarama.StringEncoder(event.Key),
		Value: sarama.StringEncoder(event.Message),
	}

	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("ошибка отправки сообщения в Kafka: %w", err)
	}

	log.Printf("Сообщение успешно отправлено в Kafka: partition=%d, offset=%d", partition, offset)
	return nil
}
