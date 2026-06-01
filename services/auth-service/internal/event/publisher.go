package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/domain"
)

// =============================================================
// Event Types
// =============================================================

const (
	ExchangeName = "auth.events"
	ExchangeType = "topic"

	EventUserRegistered   = "user.registered"
	EventUserLoggedIn     = "user.logged_in"
	EventUserLoggedOut    = "user.logged_out"
	EventUserGoogleLinked = "user.google_linked"
)

// =============================================================
// Event Payloads
// =============================================================

type BaseEvent struct {
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	ServiceID string    `json:"service_id"`
}

type UserRegisteredEvent struct {
	BaseEvent
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

type UserLoggedInEvent struct {
	BaseEvent
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	IPAddress string `json:"ip_address,omitempty"`
}

type UserLoggedOutEvent struct {
	BaseEvent
	UserID string `json:"user_id"`
}

// =============================================================
// Publisher
// =============================================================

// Publisher — mengirim events ke RabbitMQ
// Event digunakan oleh service lain (notification-service, user-service, dll)
type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewPublisher — membuat publisher dengan koneksi RabbitMQ
func NewPublisher(amqpURL string) (*Publisher, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	// Declare exchange (topic untuk routing yang fleksibel)
	err = ch.ExchangeDeclare(
		ExchangeName,
		ExchangeType,
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	return &Publisher{conn: conn, channel: ch}, nil
}

// Close — tutup koneksi RabbitMQ
func (p *Publisher) Close() {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}

// =============================================================
// Publish Methods
// =============================================================

// PublishUserRegistered — dikirim saat user baru register
// Digunakan oleh notification-service untuk kirim email verifikasi
func (p *Publisher) PublishUserRegistered(ctx context.Context, user *domain.User) error {
	event := UserRegisteredEvent{
		BaseEvent: newBaseEvent(EventUserRegistered),
		UserID:    user.ID.String(),
		Email:     user.Email,
		FullName:  user.FullName,
		Role:      string(user.Role),
	}
	return p.publish(ctx, EventUserRegistered, event)
}

// PublishUserLoggedIn — dikirim saat user berhasil login
// Digunakan untuk audit log dan anomaly detection
func (p *Publisher) PublishUserLoggedIn(ctx context.Context, user *domain.User, ipAddress string) error {
	event := UserLoggedInEvent{
		BaseEvent: newBaseEvent(EventUserLoggedIn),
		UserID:    user.ID.String(),
		Email:     user.Email,
		IPAddress: ipAddress,
	}
	return p.publish(ctx, EventUserLoggedIn, event)
}

// PublishUserLoggedOut — dikirim saat user logout
func (p *Publisher) PublishUserLoggedOut(ctx context.Context, userID string) error {
	event := UserLoggedOutEvent{
		BaseEvent: newBaseEvent(EventUserLoggedOut),
		UserID:    userID,
	}
	return p.publish(ctx, EventUserLoggedOut, event)
}

// =============================================================
// Core Publish
// =============================================================

func (p *Publisher) publish(ctx context.Context, routingKey string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return p.channel.PublishWithContext(
		ctx,
		ExchangeName, // exchange
		routingKey,   // routing key
		false,        // mandatory — jangan return error jika tidak ada consumer
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // survive broker restart
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
}

func newBaseEvent(eventType string) BaseEvent {
	return BaseEvent{
		EventType: eventType,
		Timestamp: time.Now().UTC(),
		ServiceID: "auth-service",
	}
}

// =============================================================
// NoOp Publisher — untuk testing atau saat RabbitMQ tidak dikonfigurasi
// =============================================================

// NoOpPublisher — publisher yang tidak melakukan apa-apa
// Digunakan saat RABBITMQ_URL kosong (development tanpa RabbitMQ)
type NoOpPublisher struct{}

func (n *NoOpPublisher) PublishUserRegistered(_ context.Context, _ *domain.User) error {
	return nil
}
func (n *NoOpPublisher) PublishUserLoggedIn(_ context.Context, _ *domain.User, _ string) error {
	return nil
}
func (n *NoOpPublisher) PublishUserLoggedOut(_ context.Context, _ string) error {
	return nil
}
func (n *NoOpPublisher) Close() {}

// EventPublisher — interface yang diimplementasi Publisher dan NoOpPublisher
type EventPublisher interface {
	PublishUserRegistered(ctx context.Context, user *domain.User) error
	PublishUserLoggedIn(ctx context.Context, user *domain.User, ipAddress string) error
	PublishUserLoggedOut(ctx context.Context, userID string) error
	Close()
}
