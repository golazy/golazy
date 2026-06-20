package lazymailer

import (
	"context"
	"fmt"
	"net/smtp"
	"sync"
)

type Delivery interface {
	Deliver(context.Context, Message) error
}

type DeliveryFunc func(context.Context, Message) error

func (f DeliveryFunc) Deliver(ctx context.Context, message Message) error {
	return f(ctx, message)
}

type Registry struct {
	defaultName string
	deliveries  map[string]Delivery
}

func NewRegistry(defaultName string, deliveries map[string]Delivery) *Registry {
	copied := make(map[string]Delivery, len(deliveries))
	for name, delivery := range deliveries {
		if delivery != nil {
			copied[name] = delivery
		}
	}
	return &Registry{defaultName: defaultName, deliveries: copied}
}

func (r *Registry) Deliver(ctx context.Context, name string, message Message) error {
	if r == nil {
		return fmt.Errorf("lazymailer: delivery registry is nil")
	}
	if name == "" {
		name = r.defaultName
	}
	delivery, ok := r.deliveries[name]
	if !ok {
		return fmt.Errorf("lazymailer: delivery %q is not configured", name)
	}
	return delivery.Deliver(ctx, message)
}

type SMTPDelivery struct {
	Addr string
	Auth smtp.Auth
}

func (d SMTPDelivery) Deliver(_ context.Context, message Message) error {
	raw, err := message.Bytes()
	if err != nil {
		return err
	}
	return smtp.SendMail(d.Addr, d.Auth, message.From.Address, Recipients(message.To), raw)
}

type MemoryDelivery struct {
	mu       sync.Mutex
	messages []Message
}

func (d *MemoryDelivery) Deliver(_ context.Context, message Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, message)
	return nil
}

func (d *MemoryDelivery) Messages() []Message {
	d.mu.Lock()
	defer d.mu.Unlock()
	messages := make([]Message, len(d.messages))
	copy(messages, d.messages)
	return messages
}
