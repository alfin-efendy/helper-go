package token

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/alfin-efendy/helper-go/otel"
	"github.com/golang-jwt/jwt/v5"
)

// private function parses PEM encoded private key from string
// this function is created to support both PKCS1 and PKCS8
// because jwt-go only supports PKCS1 or PKCS8, but function jwt.ParseRSAPrivateKeyFromPEM is not converting to PKCS1 or PKCS8
func parsePrivateKey(ctx context.Context, jwtSecretKey string) (*rsa.PrivateKey, error) {
	ctx, span := otel.Trace(ctx)
	defer span.End()

	// Decode private key string to PEM
	block, _ := pem.Decode([]byte(jwtSecretKey))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the private key")
	}

	// Parse private key from PEM to PKCS1
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// If fail to PKCS1 Parse private key from PEM to PKCS8
		keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		// Check if key is RSA
		var ok bool
		key, ok = keyInterface.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("failed to parse private key")
		}
	}

	return key, nil
}

// JWTSign is the function to sign JWT
func jwtSign(ctx context.Context, claims jwt.RegisteredClaims, secretPrivateKey string) (string, error) {
	ctx, span := otel.Trace(ctx)
	defer span.End()

	// Decode base64 encoded private key
	decodedPrivateKey, err := base64.StdEncoding.DecodeString(secretPrivateKey)
	if err != nil {
		return "", fmt.Errorf("could not decode token private key: %w", err)
	}

	// Parse private key from PEM
	privateKey, err := parsePrivateKey(ctx, string(decodedPrivateKey))
	if err != nil {
		return "", fmt.Errorf("could not parse token private key: %w", err)
	}

	// Sign token
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("create: sign token: %w", err)
	}

	return tokenString, nil
}

// JWTVerify is the function to verify JWT
func jwtVerify(ctx context.Context, tokenString string, secretPublicKey string) (*jwt.RegisteredClaims, error) {
	ctx, span := otel.Trace(ctx)
	defer span.End()

	// Decode base64 encoded public key
	decodedPublicKey, err := base64.StdEncoding.DecodeString(secretPublicKey)
	if err != nil {
		return nil, fmt.Errorf("could not decode: %w", err)
	}

	// Parse public key from PEM
	key, err := jwt.ParseRSAPublicKeyFromPEM(decodedPublicKey)

	if err != nil {
		return nil, fmt.Errorf("validate: parse key: %w", err)
	}

	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})

	if err != nil {
		return claims, err
	}

	if !token.Valid {
		return claims, err
	}

	return claims, nil
}
