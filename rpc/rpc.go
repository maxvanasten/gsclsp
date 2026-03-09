package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func EncodeMessage(msg any) (string, error) {
	content, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal rpc message: %w", err)
	}

	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(content), content), nil
}

type BaseMessage struct {
	Method string `json:"method"`
}

func DecodeMessage(msg []byte) (string, []byte, error) {
	header, content, found := bytes.Cut(msg, []byte{'\r', '\n', '\r', '\n'})
	if !found {
		return "", nil, errors.New("did not find separator")
	}

	contentLength, err := contentLengthFromHeader(header)
	if err != nil {
		return "", nil, err
	}
	if len(content) < contentLength {
		return "", nil, fmt.Errorf("content shorter than content-length: have %d want %d", len(content), contentLength)
	}

	var baseMessage BaseMessage
	if err := json.Unmarshal(content[:contentLength], &baseMessage); err != nil {
		return "", nil, err
	}

	return baseMessage.Method, content[:contentLength], nil
}

func Split(data []byte, _ bool) (advance int, token []byte, err error) {
	header, content, found := bytes.Cut(data, []byte{'\r', '\n', '\r', '\n'})
	if !found {
		return 0, nil, nil
	}

	contentLength, err := contentLengthFromHeader(header)
	if err != nil {
		return 0, nil, err
	}

	if len(content) < contentLength {
		return 0, nil, nil
	}

	totalLength := len(header) + 4 + contentLength

	return totalLength, data[:totalLength], nil
}

func contentLengthFromHeader(header []byte) (int, error) {
	if len(header) == 0 {
		return 0, errors.New("empty header")
	}

	lines := strings.Split(string(header), "\r\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(parts[0]), "Content-Length") {
			continue
		}

		value := strings.TrimSpace(parts[1])
		contentLength, err := strconv.Atoi(value)
		if err != nil {
			return 0, fmt.Errorf("invalid Content-Length %q: %w", value, err)
		}
		if contentLength < 0 {
			return 0, fmt.Errorf("negative Content-Length: %d", contentLength)
		}
		return contentLength, nil
	}

	return 0, errors.New("missing Content-Length header")
}
