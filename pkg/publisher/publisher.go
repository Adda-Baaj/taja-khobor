package publisher

// Package publisher contains logic to publish events to SQS/Kafka/etc.

// Publish sends an event to the configured sink.
func Publish(topic string, payload []byte) error {
	// TODO: implement SQS/Kafka publishing
	return nil
}
