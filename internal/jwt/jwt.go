package jwt

import (
	"fmt"
	"strings"
	"time"

	"github.com/DevDashkovsky/room-booking/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

func GenerateToken(userID, role, secret string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(tokenString, secret string) (string, string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		return "", "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", fmt.Errorf("invalid claims")
	}

	userID, _ := claims["user_id"].(string)
	role, _ := claims["role"].(string)

	if !domain.ValidUUID(userID) || (role != "admin" && role != "user") {
		return "", "", fmt.Errorf("invalid user_id or role")
	}
	return strings.ToLower(userID), role, nil
}
