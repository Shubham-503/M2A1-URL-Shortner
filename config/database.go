package config

import (
	"M2A1-URL-Shortner/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

var DB *gorm.DB

func InitDB() error {
	var err error
	// Open SQLite database with GORM
	DB, err = gorm.Open(sqlite.Open("url_shortener.db"), &gorm.Config{})
	if err != nil {
		return err
	}

	// Auto migrate the schema
	err = DB.AutoMigrate(&models.URLShortener{}, &models.User{})
	if err != nil {
		return err
	}

	return nil
}
