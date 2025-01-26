package token

import (
	"context"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/database"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// signToken is a helper private function that signs a JWT token and stores it in Redis.
func signToken(ctx context.Context, id, issuer, subject, keyPrivate string, expiredHour int, ability []string) (string, time.Time, error) {
	// Generate token expired time
	tokenExpiredDuration := time.Duration(expiredHour) * time.Hour
	tokenExpired := time.Now().UTC().Add(tokenExpiredDuration)

	// Generate JWT token
	token, err := jwtSign(
		jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   subject,
			Audience:  ability,
			ExpiresAt: jwt.NewNumericDate(tokenExpired),
			NotBefore: jwt.NewNumericDate(time.Now().UTC()),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ID:        id,
		},
		keyPrivate,
	)

	if err != nil {
		return "", time.Time{}, err
	}

	redis := database.GetRedisClient()

	// Store token to Redis
	if err := redis.Set(ctx, subject, id, tokenExpiredDuration).Err(); err != nil {
		return "", time.Time{}, err
	}

	return token, tokenExpired, nil
}

// TokenGenerate is a helper private function that generates access and refresh tokens.
func TokenGenerate(ctx context.Context, subject string, ability []string) (string, time.Time, string, time.Time, error) {
	// Generate access token id
	accessId := uuid.New().String()
	config := config.Config
	issuer := config.App.Name
	accessExpired := config.Token.AccessExpireHour
	accessPrivateKey := config.Token.AccessPrivateKey

	// Generate access token
	accessToken, accessTokenExpired, err := signToken(ctx, accessId, issuer, subject, accessPrivateKey, accessExpired, ability)

	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	// Generate refresh token id
	refreshId := uuid.New().String()
	refreshExpired := config.Token.RefreshExpireHour
	refreshPrivateKey := config.Token.RefreshPrivateKey

	// Generate refresh token
	refreshToken, refreshTokenExpired, err := signToken(ctx, refreshId, issuer, accessId, refreshPrivateKey, refreshExpired, []string{})

	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	return accessToken, accessTokenExpired, refreshToken, refreshTokenExpired, nil
}

// TokenValidation is a helper private function that validates a JWT token.
// param tokenType is access or refresh.
func TokenValidation(ctx context.Context, token, tokenType string, validationExpired bool) (*jwt.RegisteredClaims, error) {
	var keyPublic string
	if tokenType == "access" {
		keyPublic = config.Config.Token.AccessPublicKey
	} else {
		keyPublic = config.Config.Token.RefreshPublicKey
	}

	// Validate access token
	dataToken, err := jwtVerify(token, keyPublic)

	if err != nil {
		if !validationExpired && err.Error() == "token has invalid claims: token is expired" {
			return dataToken, nil
		}
		return nil, err
	}

	redis := database.GetRedisClient()

	// Check refresh token in Redis
	if err := redis.Get(ctx, dataToken.Subject).Err(); err != nil {
		return nil, err
	}

	return dataToken, nil
}
