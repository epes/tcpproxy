package tcpproxy

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server interface {
	Serve()
}

type ServerConfig struct {
	Port                  int
	SourceHeaderByteCount int
	SourceHeaderDeadline  time.Duration
	WelcomeMiddleware     func(WelcomeBytes []byte) (DestinationConfig, error)
}

type DestinationConfig struct {
	Address string
	Header  []byte
}

type server struct {
	cfg    ServerConfig
	logger Logger
}

func New(logger Logger, cfg ServerConfig) Server {
	return &server{logger: logger, cfg: cfg}
}

func (s *server) Serve() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		s.logger.Fatal(fmt.Errorf("starting proxy server: %w", err))
		return
	}

	s.logger.Info(fmt.Sprintf("tcp proxy listening on port :%d", s.cfg.Port))

	for {
		c, err := listener.Accept()
		if err != nil {
			s.logger.Error(fmt.Errorf("error accepting tcp proxy source connection: %w", err))
			continue
		}
		go s.handleSource(c)
	}
}

func (s *server) handleSource(source net.Conn) {
	defer source.Close()

	source.SetReadDeadline(time.Now().Add(s.cfg.SourceHeaderDeadline))

	sourceHeader := make([]byte, s.cfg.SourceHeaderByteCount)
	if _, err := io.ReadFull(source, sourceHeader); err != nil {
		s.logger.Error(fmt.Errorf("reading source header: %w", err))
		return
	}

	destinationCfg, err := s.cfg.WelcomeMiddleware(sourceHeader)
	if err != nil {
		s.logger.Error(fmt.Errorf("failed welcome middleware: %w", err))
		return
	}

	destination, err := net.Dial("tcp", destinationCfg.Address)
	if err != nil {
		s.logger.Error(fmt.Errorf("connecting to destination server: %w", err))
		return
	}
	defer destination.Close()

	if _, err := destination.Write(destinationCfg.Header); err != nil {
		s.logger.Error(fmt.Errorf("writing destination header: %w", err))
		return
	}

	source.SetDeadline(time.Time{})
	destination.SetDeadline(time.Time{})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		forward(source, destination)
	}()

	go func() {
		defer wg.Done()
		forward(destination, source)
	}()

	wg.Wait()
}

func forward(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	io.Copy(src, dest)
}
