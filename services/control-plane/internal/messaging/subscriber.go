package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/sette/guardian-lan/services/control-plane/internal/domain"
	"github.com/sette/guardian-lan/services/control-plane/internal/service"
)

const (
	DeviceDiscoveredSubject = "network.device.discovered"
	DeviceUpdatedSubject    = "network.device.updated"
	DNSObservedSubject      = "network.dns.query_observed"
	FlowObservedSubject     = "network.flow.observed"
)

type Subscriber struct {
	conn         *nats.Conn
	orchestrator *service.Orchestrator
}

func NewSubscriber(conn *nats.Conn, orchestrator *service.Orchestrator) *Subscriber {
	return &Subscriber{
		conn:         conn,
		orchestrator: orchestrator,
	}
}

func (s *Subscriber) Start(ctx context.Context) error {
	handlers := map[string]func(context.Context, []byte) error{
		DeviceDiscoveredSubject: func(ctx context.Context, payload []byte) error {
			var event domain.DeviceEvent
			if err := json.Unmarshal(payload, &event); err != nil {
				return fmt.Errorf("unmarshal device discovered event: %w", err)
			}
			return s.orchestrator.HandleDeviceEvent(ctx, event, true)
		},
		DeviceUpdatedSubject: func(ctx context.Context, payload []byte) error {
			var event domain.DeviceEvent
			if err := json.Unmarshal(payload, &event); err != nil {
				return fmt.Errorf("unmarshal device updated event: %w", err)
			}
			return s.orchestrator.HandleDeviceEvent(ctx, event, false)
		},
		DNSObservedSubject: func(ctx context.Context, payload []byte) error {
			var event domain.DNSEvent
			if err := json.Unmarshal(payload, &event); err != nil {
				return fmt.Errorf("unmarshal dns event: %w", err)
			}
			return s.orchestrator.HandleDNSEvent(ctx, event)
		},
		FlowObservedSubject: func(ctx context.Context, payload []byte) error {
			var event domain.FlowEvent
			if err := json.Unmarshal(payload, &event); err != nil {
				return fmt.Errorf("unmarshal flow event: %w", err)
			}
			return s.orchestrator.HandleFlowEvent(ctx, event)
		},
	}

	for subject, handler := range handlers {
		subject := subject
		handler := handler
		if _, err := s.conn.Subscribe(subject, func(msg *nats.Msg) {
			if err := handler(ctx, msg.Data); err != nil {
				log.Printf("subject=%s error=%v", subject, err)
			}
		}); err != nil {
			return fmt.Errorf("subscribe %s: %w", subject, err)
		}
	}

	if err := s.conn.Flush(); err != nil {
		return fmt.Errorf("flush nats subscriptions: %w", err)
	}

	go func() {
		<-ctx.Done()
		if err := s.conn.Drain(); err != nil {
			log.Printf("drain nats connection: %v", err)
		}
	}()

	return nil
}
