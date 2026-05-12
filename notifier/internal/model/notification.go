package model

import "time"

type NotificationType string

const (
	TypeEmail NotificationType = "email"
	TypePush  NotificationType = "push"
)

type NotificationStatus string

const (
	StatusPending NotificationStatus = "pending"
	StatusSent    NotificationStatus = "sent"
	StatusFailed  NotificationStatus = "failed"
)

// IDs are UUID strings (matching order-service). Kept as plain strings here
// because the notifier only logs/renders them — no need to pull in a UUID lib.
type Notification struct {
	ID      string
	OrderID string
	UserID  string
	Type    NotificationType
	Status  NotificationStatus
	SentAt  time.Time

	// Subject and Body are pre-rendered by the service layer based on the
	// triggering event. The Sender just delivers them — it doesn't need to
	// know what kind of event led to this notification.
	Subject string
	Body    string
}
