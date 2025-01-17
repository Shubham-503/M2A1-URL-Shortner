package models

import "time"

type User struct {
	ID        uint `gorm:"primaryKey;autoIncrement"`
	Email     string
	Name      string
	ApiKey    string `gorm:"unique"`
	Tier      string `gorm:"default:'hobby';check: tier IN ('hobby', 'enterprise')"`
	CreatedAt time.Time
}
