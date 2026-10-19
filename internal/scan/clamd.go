package scan

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type Verdict uint8

const (
	Clean Verdict = iota + 1
	Infected
)

type Result struct {
	Verdict   Verdict
	Signature string
}

type Scanner interface {
	Scan(context.Context, io.Reader) (Result, error)
}

var (
	ErrProtocol = errors.New("invalid clamd response")
	ErrDaemon   = errors.New("clamd scan failed")
	ErrTooLarge = errors.New("scan stream exceeds the maximum size")
)

type Client struct {
	address  string
	timeout  time.Duration
	maxBytes int64
}

func NewClient(address string, timeout time.Duration, maxBytes int64) (*Client, error) {
	_, port, err := net.SplitHostPort(address)
	number, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || number < 1 || number > 65535 || timeout <= 0 || maxBytes <= 0 {
		return nil, errors.New("clamd host:port, positive timeout, and positive size limit are required")
	}
	return &Client{address, timeout, maxBytes}, nil
}

// Scan uses one connection per file. Cancellation closes the connection and any
// io.Closer source to interrupt network reads. Other readers must not block forever.
func (c *Client) Scan(ctx context.Context, source io.Reader) (Result, error) {
	if source == nil {
		return Result{}, errors.New("scan reader is required")
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", c.address)
	if err != nil {
		return Result{}, err
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() {
		_ = conn.Close()
		if closer, ok := source.(io.Closer); ok {
			_ = closer.Close()
		}
	})
	defer stop()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return Result{}, err
	}
	if err := writeAll(conn, []byte("zINSTREAM\x00")); err != nil {
		return Result{}, err
	}
	if err := sendStream(ctx, conn, source, c.maxBytes); err != nil {
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		return Result{}, err
	}
	reply, err := bufio.NewReaderSize(conn, 4096).ReadSlice(0)
	if err != nil {
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		return Result{}, fmt.Errorf("%w: %w", ErrProtocol, err)
	}
	return parseReply(string(reply[:len(reply)-1]))
}

func parseReply(reply string) (Result, error) {
	if reply == "stream: OK" {
		return Result{Verdict: Clean}, nil
	}
	if strings.HasPrefix(reply, "stream: ") && strings.HasSuffix(reply, " FOUND") {
		name := strings.TrimSuffix(strings.TrimPrefix(reply, "stream: "), " FOUND")
		if name != "" && !strings.ContainsAny(name, "\r\n\x00") {
			return Result{Infected, name}, nil
		}
	}
	if strings.HasSuffix(reply, " ERROR") {
		return Result{}, fmt.Errorf("%w: %s", ErrDaemon, reply)
	}
	return Result{}, ErrProtocol
}
