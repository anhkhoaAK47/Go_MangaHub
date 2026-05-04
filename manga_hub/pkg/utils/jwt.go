package utils

import (
	"encoding/json"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
)


func GenerateJWT(userID string, username string, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"username": username,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString([]byte(secret))
}

func LoadUserToken() (string, string) {
	data, err := os.ReadFile(".token");
	if err != nil {
		return "", ""
	}

	var token map[string]string
	if err := json.Unmarshal(data, &token); err != nil {
		return "", ""
	}

	return token["user_id"], token["exp"]
}