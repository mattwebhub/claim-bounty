package scanner

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type ClamAV struct {
	address string
	timeout time.Duration
}

func NewClamAV(address string, timeout time.Duration) (*ClamAV, error) {
	if strings.TrimSpace(address) == "" {
		return nil, errors.New("scanner: ClamAV address is required")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &ClamAV{address: address, timeout: timeout}, nil
}

func (scanner *ClamAV) Inspect(ctx context.Context, source io.Reader, expectedSize int64, _ string) (string, bool, string, error) {
	dialer := net.Dialer{Timeout: scanner.timeout}
	connection, err := dialer.DialContext(ctx, "tcp", scanner.address)
	if err != nil {
		return "", false, "", errors.New("scanner: ClamAV unavailable")
	}
	defer connection.Close()
	deadline := time.Now().Add(scanner.timeout)
	_ = connection.SetDeadline(deadline)
	if _, err := connection.Write([]byte("zINSTREAM\x00")); err != nil {
		return "", false, "", errors.New("scanner: request failed")
	}
	buffer := make([]byte, 32*1024)
	header := make([]byte, 0, 512)
	var total int64
	for {
		n, readErr := source.Read(buffer)
		if n > 0 {
			total += int64(n)
			if total > expectedSize {
				return "", false, "size_mismatch", nil
			}
			if len(header) < 512 {
				take := min(n, 512-len(header))
				header = append(header, buffer[:take]...)
			}
			var size [4]byte
			if uint64(n) > uint64(^uint32(0)) {
				return "", false, "", errors.New("scanner: chunk is too large")
			}
			binary.BigEndian.PutUint32(size[:], uint32(n))
			if _, err := connection.Write(size[:]); err != nil {
				return "", false, "", errors.New("scanner: request failed")
			}
			if _, err := connection.Write(buffer[:n]); err != nil {
				return "", false, "", errors.New("scanner: request failed")
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return "", false, "", errors.New("scanner: source read failed")
		}
	}
	if total != expectedSize {
		return "", false, "size_mismatch", nil
	}
	if _, err := connection.Write([]byte{0, 0, 0, 0}); err != nil {
		return "", false, "", errors.New("scanner: request failed")
	}
	response, err := bufio.NewReader(connection).ReadString(0)
	if err != nil {
		return "", false, "", errors.New("scanner: response failed")
	}
	mediaType := http.DetectContentType(header)
	switch {
	case strings.Contains(response, " OK"):
		return mediaType, true, "", nil
	case strings.Contains(response, " FOUND"):
		return mediaType, false, "malware_detected", nil
	default:
		return mediaType, false, "", errors.New("scanner: ClamAV returned an indeterminate result")
	}
}
