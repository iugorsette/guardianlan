package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/sette/guardian-lan/services/control-plane/internal/api"
	"github.com/sette/guardian-lan/services/control-plane/internal/config"
	"github.com/sette/guardian-lan/services/control-plane/internal/messaging"
	"github.com/sette/guardian-lan/services/control-plane/internal/repository"
	"github.com/sette/guardian-lan/services/control-plane/internal/service"
)

func Run(ctx context.Context) error {
	cfg := config.Load()

	store, err := repository.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	natsConn, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		return fmt.Errorf("connect to nats: %w", err)
	}
	defer natsConn.Close()

	orchestrator := service.NewOrchestrator(store, messaging.NewNATSPublisher(natsConn), cfg.ExpectedDNSResolver)
	subscriber := messaging.NewSubscriber(natsConn, orchestrator)
	if err := subscriber.Start(ctx); err != nil {
		return err
	}

	server := api.NewServer(cfg.HTTPAddr, store, orchestrator)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown server: %v", err)
		}
	}()

	log.Printf("control-plane listening on %s", cfg.HTTPAddr)
	return server.ListenAndServe()
}
