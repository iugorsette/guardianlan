package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/sette/guardian-lan/services/control-plane/internal/domain"
)

const AlertRaisedSubject = "network.alert.raised"

type AlertPublisher interface {
	PublishAlert(context.Context, domain.Alert) error
}

type NATSPublisher struct {
	conn *nats.Conn
}

func NewNATSPublisher(conn *nats.Conn) *NATSPublisher {
	return &NATSPublisher{conn: conn}
}

func (p *NATSPublisher) PublishAlert(_ context.Context, alert domain.Alert) error {
	payload, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("marshal alert: %w", err)
	}

	if err := p.conn.Publish(AlertRaisedSubject, payload); err != nil {
		return fmt.Errorf("publish alert: %w", err)
	}

	return nil
}

type NoopPublisher struct{}

func (NoopPublisher) PublishAlert(context.Context, domain.Alert) error {
	return nil
}
