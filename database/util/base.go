package util

import (
	"time"
)

// This model is common table structure

// This struct is for record time of created and updated
type CommonTime struct {
	CreatedAt time.Time `gorm:"not null; default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time
}

// This struct is for if the table need to use soft delete
type SoftDelete struct {
	DeletedBy *string    `sql:"index"`
	DeletedAt *time.Time `sql:"index"`
}

// This struct is for if the table need to use soft delete and record time of created and updated
type CommonBy struct {
	CreatedBy string `gorm:"not null"`
	UpdatedBy string
}
