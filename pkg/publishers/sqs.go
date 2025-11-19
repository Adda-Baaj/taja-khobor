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

// sqsClient defines the minimal subset of the SQS client used by sqsPublisher.
type sqsClient interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

// sqsPublisher implements the Publisher interface for AWS SQS.
type sqsPublisher struct {
	id       string
	queueURL string
	typ      string
	client   sqsClient
	log      Logger
}

// newSQSPublisher creates a new SQS publisher with the given configuration.
func newSQSPublisher(ctx context.Context, cfg PublisherConfig, log Logger) (Publisher, error) {
	if cfg.SQS == nil {
		return nil, fmt.Errorf("publisher %q missing sqs configuration", cfg.ID)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	awsCfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(cfg.SQS.Region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &sqsPublisher{
		id:       cfg.ID,
		typ:      TypeSQS,
		queueURL: cfg.SQS.QueueURL,
		client:   sqs.NewFromConfig(awsCfg),
		log:      ensureLogger(log),
	}, nil
}

func (s *sqsPublisher) ID() string   { return s.id }
func (s *sqsPublisher) Type() string { return s.typ }

// Publish sends the event to the configured SQS queue.
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
		s.log.ErrorObj("sqs publisher send failed", "publisher_sqs_error", map[string]any{
			"publisher_id": s.id,
			"error":        err.Error(),
		})
		return fmt.Errorf("send message to sqs: %w", err)
	}
	s.log.DebugObj("sqs publisher delivered event", "publisher_sqs_delivery", map[string]any{
		"publisher_id": s.id,
	})
	return nil
}
