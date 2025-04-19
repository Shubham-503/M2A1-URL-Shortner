package utils

import (
	"M2A1-URL-Shortner/config"
	"M2A1-URL-Shortner/models"
	"bytes"
	"fmt"
	"os"

	"github.com/disintegration/imaging"
)

func CheckThumbnail() {
	fmt.Println("Check Thumbnail called")
	var users []models.User
	result := config.DB.Find(&users)
	if result.Error != nil {
		fmt.Println("error fetching users")
	}
	// profileImgBytes, err := os.ReadFile("../assets/image/profileImg.jpg")
	profileImgBytes, err := os.ReadFile("assets/image/profileImg.jpg")
	if err != nil {
		fmt.Printf("Error in profilepc read %v", err.Error())
	}

	users[0].ProfileImg = &profileImgBytes
	// fmt.Printf("user at zero index %+v", users[0])
	// fmt.Println(users[0].ProfileImg)
	for _, user := range users {
		// fmt.Printf("user %+v\n", user)
		if user.ProfileImg != nil && user.Thumbnail == nil {
			fmt.Println("Thumbnail need to be updated")
			img, err := imaging.Decode(bytes.NewReader(*user.ProfileImg))
			if err != nil {
				fmt.Println("Failed to decode profile image:", err)
				continue
			}

			// Resize to 300x300
			resized := imaging.Resize(img, 300, 300, imaging.Lanczos)

			var buf bytes.Buffer
			err = imaging.Encode(&buf, resized, imaging.JPEG)
			if err != nil {
				fmt.Println("Failed to encode resized image:", err)
				continue
			}
			thumbBytes := buf.Bytes()
			user.Thumbnail = &thumbBytes
			config.DB.Save(&user)
		}

	}

}
