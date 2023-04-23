package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/epes/tcpproxy"
	"go.uber.org/zap"
	"time"
)

func main() {
	logger := zap.NewExample()
	defer logger.Sync()

	cfg := tcpproxy.ServerConfig{
		Port:                  8888,
		SourceHeaderByteCount: 66, // 64 bytes for token in hex + 2 bytes to select the destination
		SourceHeaderDeadline:  time.Duration(60 * time.Second),
		WelcomeMiddleware:     welcomeMiddleware,
	}

	proxy := tcpproxy.New(logger.Sugar(), cfg)

	logger.Debug(fmt.Sprintf("ADMIN TOKEN: %s", hex.EncodeToString(AdminToken)))
	logger.Debug(fmt.Sprintf("NORMAL TOKEN: %s", hex.EncodeToString(NormalToken)))

	proxy.Serve()
}

func welcomeMiddleware(header []byte) (tcpproxy.DestinationConfig, error) {
	token, err := hex.DecodeString(string(header[:64]))
	if err != nil {
		panic(err)
	}
	destination := string(header[64:])

	userID, err := authenticate(token, destination)
	if err != nil {
		return tcpproxy.DestinationConfig{}, fmt.Errorf("auth: %w", err)
	}

	destinationAddress, err := address(destination)
	if err != nil {
		return tcpproxy.DestinationConfig{}, fmt.Errorf("resolving destination %w", err)
	}

	return tcpproxy.DestinationConfig{
		Address: destinationAddress,
		Header:  userID,
	}, nil
}

var (
	AdminToken   = key()
	AdminUserId  = "admin user"
	NormalToken  = key()
	NormalUserId = "normal user"
)

// this is the wrong order of things but just an example
// - first check for valid token and fail immediately if invalid
func authenticate(token []byte, destination string) ([]byte, error) {
	// only admins can access the "00" server
	if destination == "00" {
		if !bytes.Equal(token, AdminToken) {
			return nil, fmt.Errorf("non admin attempting to access admin server")
		}
	}

	if bytes.Equal(token, AdminToken) {
		return []byte(AdminUserId), nil
	}

	if bytes.Equal(token, NormalToken) {
		return []byte(NormalUserId), nil
	}

	return nil, fmt.Errorf("invalid token")
}

func address(destination string) (string, error) {
	if destination == "00" {
		return "localhost:4444", nil
	}

	return "localhost:5555", nil
}

func key() []byte {
	token := make([]byte, 32)
	_, err := rand.Read(token)
	if err != nil {
		panic(err)
	}

	return token
}
