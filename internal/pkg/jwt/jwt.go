package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

func NewToken(user domain.User, ttl time.Duration, secret []byte) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["uuid"] = user.Uuid
	claims["login"] = user.Login
	claims["exp"] = time.Now().Add(ttl).Unix()

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateToken(tokenString string, secret []byte) (bool, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Проверка метода подписи, ожидаемого вашим приложением
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return false, err
	}

	if !token.Valid {
		return false, fmt.Errorf("token is invalid")
	}

	return true, nil
}

func GetUserUuidFromToken(tokenString string, secret []byte) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Проверка метода подписи, ожидаемого вашим приложением
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("token parsing failed: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return uuid.Nil, errors.New("error parsing token")
	}
	uuidString, ok := claims["uuid"].(string)

	uuidUser, err := uuid.Parse(uuidString)
	if err != nil || !ok {
		return uuid.Nil, fmt.Errorf("token parsing failed: %w", err)
	}
	return uuidUser, nil
}
