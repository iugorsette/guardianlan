package repository

import (
	"context"
	"errors"

	"github.com/sette/guardian-lan/services/control-plane/internal/domain"
)

var ErrAlreadyExists = errors.New("record already exists")

type Store interface {
	UpsertDevice(context.Context, domain.Device) (domain.Device, bool, error)
	GetDevice(context.Context, string) (domain.Device, error)
	ListDevices(context.Context) ([]domain.Device, error)
	UpdateDeviceProfile(context.Context, string, string) (domain.Device, error)
	StoreDNSEvent(context.Context, domain.DNSEvent) error
	ListDNSEvents(context.Context, int) ([]domain.DNSEvent, error)
	StoreFlowEvent(context.Context, domain.FlowEvent) error
	ListFlowEvents(context.Context, int) ([]domain.FlowEvent, error)
	StoreObservation(context.Context, domain.Observation) error
	CreateAlert(context.Context, domain.Alert) error
	ListAlerts(context.Context, int, string) ([]domain.Alert, error)
	AckAlert(context.Context, string) (domain.Alert, error)
}
