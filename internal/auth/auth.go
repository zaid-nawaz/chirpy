package auth

import (
	"log"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"

	"time"
	"github.com/golang-jwt/jwt/v5"
	"errors"
	"strings"
	"net/http"
	"crypto/rand"
	"encoding/hex"
)

func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		log.Fatal(err)
	}

	return hash, err
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		log.Fatal(err)
	}

	return match, err
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {

	type MyCustomClaims struct {
		Foo string `json:"foo"`
		jwt.RegisteredClaims
	}

	claims := MyCustomClaims{
		"bar",
		jwt.RegisteredClaims{
			Issuer : "test",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			Subject:   userID.String(),		
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	ss, err := token.SignedString([]byte(tokenSecret))
	return ss, err

}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error ) {

	type MyCustomClaims struct {
		Foo string `json:"foo"`
		jwt.RegisteredClaims
	}

	token, err := jwt.ParseWithClaims(tokenString, &MyCustomClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return uuid.Nil, err
	} 

	claims, ok := token.Claims.(*MyCustomClaims)
	if !ok {
		return uuid.Nil, errors.New("invalid claims")
	}

	if !token.Valid {
		return uuid.Nil, errors.New("token is invalid")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
    authHeader := headers.Get("Authorization")
    
    if authHeader == "" {
        return "", errors.New("no authorization header")
    }

    tokenString := strings.TrimPrefix(authHeader, "Bearer ")

    if tokenString == authHeader {
        return "", errors.New("authorization header is not a bearer token")
    }

    return tokenString, nil
}

func MakeRefreshToken() string {

	key := make([]byte, 32)
	rand.Read(key)

	encodedStr := hex.EncodeToString(key)

	return encodedStr

}

func GetAPIKey(headers http.Header) (string, error) {
    authHeader := headers.Get("Authorization")
    
    if authHeader == "" {
        return "", errors.New("no authorization header")
    }

    tokenString := strings.TrimPrefix(authHeader, "ApiKey ")

    if tokenString == authHeader {
        return "", errors.New("authorization header is not a bearer token")
    }

    return tokenString, nil
}