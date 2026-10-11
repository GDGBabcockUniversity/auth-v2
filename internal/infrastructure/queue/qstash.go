package queue

import (
	"log/slog"

	"github.com/upstash/qstash-go"
)

type QStash struct {
	Client *qstash.Client
	Queue  *qstash.Queue
}

func NewClient(token string) *QStash {
	client := qstash.NewClient(token)

	queueName := "users"

	err := client.Queues().Upsert(qstash.Queue{
		Name:        queueName,
		Parallelism: 1,
	})

	if err != nil {
		slog.Error("Error upserting queue", "ERROR", err)
		return nil
	}

	return &QStash{
		Client: client,
	}
}

func (q *QStash) Publish(urlGroup string, body map[string]any) error {

	_, err := q.Client.UrlGroups().PublishJSON(qstash.PublishUrlGroupJSONOptions{
		UrlGroup: urlGroup,
		Body:     body,
	})

	if err != nil {
		return err
	}

	slog.Info("Event published")

	return nil
}
