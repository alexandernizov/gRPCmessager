package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("token is invalid")
)

func NewTokens(user domain.User, accessTtl time.Duration, refreshTtl time.Duration, secret []byte) (domain.Tokens, error) {
	accessToken := jwt.New(jwt.SigningMethodHS256)
	accessClaims := accessToken.Claims.(jwt.MapClaims)
	accessClaims["uuid"] = user.Uuid
	accessClaims["login"] = user.Login
	accessClaims["expired"] = time.Now().Add(accessTtl).Unix()
	accessString, err := accessToken.SignedString(secret)
	if err != nil {
		return domain.Tokens{}, err
	}

	refreshToken := jwt.New(jwt.SigningMethodHS256)
	refreshClaims := refreshToken.Claims.(jwt.MapClaims)
	refreshClaims["uuid"] = user.Uuid
	refreshClaims["login"] = user.Login
	refreshClaims["expired"] = time.Now().Add(refreshTtl).Unix()
	refreshString, err := refreshToken.SignedString(secret)
	if err != nil {
		return domain.Tokens{}, err
	}

	return domain.Tokens{AccessToken: accessString, RefreshToken: refreshString}, nil
}

func ValidateToken(tokenString string, secret []byte) (bool, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Check signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	//Common check
	if !token.Valid {
		return false, ErrInvalidToken
	}

	//Check if expired
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, ErrInvalidToken
	}
	expired, ok := claims["expired"].(float64)
	if !ok {
		return false, ErrInvalidToken
	}

	if expired < float64(time.Now().Unix()) {
		return false, ErrInvalidToken
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
