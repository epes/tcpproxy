package main

import (
	"fmt"
	"github.com/epes/tcpproxy"
	"go.uber.org/zap"
	"os"
	"strconv"
	"time"
)

func main() {
	logger := zap.NewExample()
	defer logger.Sync()

	if len(os.Args) < 3 {
		panic("provide a proxy port and target port")
	}

	proxyPort, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(fmt.Errorf("proxy port incorrect: %w", err))
	}

	cfg := tcpproxy.ServerConfig{
		Port:                  proxyPort,
		SourceHeaderByteCount: 0,
		SourceHeaderDeadline:  time.Duration(3600 * time.Second),
		WelcomeMiddleware:     welcomeMiddleware,
	}

	proxy := tcpproxy.New(logger.Sugar(), cfg)

	proxy.Serve()
}

func welcomeMiddleware(header []byte) (tcpproxy.DestinationConfig, error) {
	destinationPort, err := strconv.Atoi(os.Args[2])
	if err != nil {
		panic(fmt.Errorf("destination port incorrect: %w", err))
	}

	return tcpproxy.DestinationConfig{
		Address: fmt.Sprintf("localhost:%d", destinationPort),
		Header:  nil,
	}, nil
}
