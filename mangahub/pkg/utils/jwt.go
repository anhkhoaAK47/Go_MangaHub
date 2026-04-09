package utils

import (
	"time"

	"github.com/golang-jwt/jwt"
)




func GenerateJWT(userID string, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": userID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString([]byte(secret))
}


// func ValidateToken(tokenString string) (Claims, error){

// }