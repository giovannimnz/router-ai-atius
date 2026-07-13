package model

import "time"

// CodexOAuthOperation persists fenced OAuth stages across process and replica loss.
type CodexOAuthOperation struct {
	OperationKey     string `gorm:"primaryKey;size:191"`
	Kind             string `gorm:"size:32;not null;index"`
	UserID           int    `gorm:"not null;index"`
	ChannelID        int    `gorm:"not null;index"`
	DeviceAuthID     string `gorm:"size:191"`
	UserCode         string `gorm:"size:64"`
	Status           string `gorm:"size:64;not null;index"`
	Stage            string `gorm:"size:32;not null"`
	Owner            string `gorm:"size:64"`
	Fence            uint64 `gorm:"not null;default:0"`
	LeaseUntil       int64  `gorm:"not null;default:0"`
	ExpiresAt        int64  `gorm:"not null;index"`
	NextAttemptAt    int64  `gorm:"not null;default:0"`
	RetryCount       int    `gorm:"not null;default:0"`
	ProtectedPayload string `gorm:"type:text"`
	GenerationHash   string `gorm:"size:64"`
	Result           string `gorm:"type:text"`
	ErrorMessage     string `gorm:"type:text"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (CodexOAuthOperation) TableName() string { return "codex_oauth_operations" }
