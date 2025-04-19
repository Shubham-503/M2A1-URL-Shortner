package models

import "time"

type User struct {
	ID         uint `gorm:"primaryKey;autoIncrement"`
	Email      string
	Name       string
	ApiKey     string  `gorm:"unique"`
	Tier       string  `gorm:"default:'hobby';check: tier IN ('hobby', 'enterprise')"`
	ProfileImg *[]byte `gorm:"type:blob"`
	Thumbnail  *[]byte `gorm:"type:blob"`
	CreatedAt  time.Time
}
