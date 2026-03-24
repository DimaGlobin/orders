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

type Notification struct {
	ID      int64
	OrderID int64
	UserID  int64
	Type    NotificationType
	Status  NotificationStatus
	SentAt  time.Time
}
