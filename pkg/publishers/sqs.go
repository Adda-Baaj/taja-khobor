package publishers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type sqsClient interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type sqsPublisher struct {
	id       string
	queueURL string
	typ      string
	client   sqsClient
}

func newSQSPublisher(cfg PublisherConfig) (Publisher, error) {
	if cfg.SQS == nil {
		return nil, fmt.Errorf("publisher %q missing sqs configuration", cfg.ID)
	}

	awsCfg, err := awscfg.LoadDefaultConfig(context.Background(), awscfg.WithRegion(cfg.SQS.Region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &sqsPublisher{
		id:       cfg.ID,
		typ:      TypeSQS,
		queueURL: cfg.SQS.QueueURL,
		client:   sqs.NewFromConfig(awsCfg),
	}, nil
}

func (s *sqsPublisher) ID() string   { return s.id }
func (s *sqsPublisher) Type() string { return s.typ }

func (s *sqsPublisher) Publish(ctx context.Context, evt Event) error {
	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.queueURL),
		MessageBody: aws.String(string(payload)),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"provider_id": {
				DataType:    aws.String("String"),
				StringValue: aws.String(evt.ProviderID),
			},
		},
	}

	if _, err := s.client.SendMessage(ctx, input); err != nil {
		return fmt.Errorf("send message to sqs: %w", err)
	}
	return nil
}
