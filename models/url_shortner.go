package models

import "time"

// Define the URLShortener model
type URLShortener struct {
	ID             uint   `gorm:"primaryKey"`
	OriginalURL    string `gorm:"size:2083;not null"`
	ShortCode      string `gorm:"unique;not null"`
	HitCount       uint   `gorm:"default:0"`
	ShortenCount   uint   `gorm:"default:1"`
	CreatedAt      time.Time
	ApiKey         string
	Password       *string `json:"password,omitempty"`
	ExpiredAt      *time.Time
	LastAccessedAt *time.Time
	DeletedAt      *time.Time
	UserID         uint
	User           User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
