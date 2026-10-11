package xray

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// ShadowsocksConfig stores ss:// URL parameters.
type ShadowsocksConfig struct {
	Method   string
	Password string
	Address  string
	Port     int
	UoT      bool
	Remark   string

	raw *url.URL `json:"-"`
}

type ShadowsocksURL struct {
	Config *ShadowsocksConfig
}

func decodeSSBase64(raw string) (string, error) {
	v, err := decodeBase64Any(raw)
	if err != nil {
		return "", err
	}
	return v, nil
}

func parseSSMethodPassword(user *url.Userinfo) (string, string, error) {
	if user == nil {
		return "", "", fmt.Errorf("missing ss userinfo")
	}

	if password, ok := user.Password(); ok {
		return strings.TrimSpace(user.Username()), strings.TrimSpace(password), nil
	}

	decoded, err := decodeSSBase64(user.Username())
	if err != nil {
		return "", "", fmt.Errorf("invalid ss credentials: %w", err)
	}
	idx := strings.Index(decoded, ":")
	if idx < 1 {
		return "", "", fmt.Errorf("invalid ss credentials format")
	}
	return strings.TrimSpace(decoded[:idx]), strings.TrimSpace(decoded[idx+1:]), nil
}

func parseSSFullEncoded(rawURL string) (method string, password string, host string, port int, err error) {
	payload := strings.TrimPrefix(rawURL, "ss://")
	payload = strings.TrimPrefix(payload, "shadowsocks://")
	if i := strings.IndexAny(payload, "?#"); i >= 0 {
		payload = payload[:i]
	}
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return "", "", "", 0, fmt.Errorf("empty ss payload")
	}

	decoded, err := decodeSSBase64(payload)
	if err != nil {
		return "", "", "", 0, fmt.Errorf("invalid encoded ss payload: %w", err)
	}

	at := strings.LastIndex(decoded, "@")
	if at < 1 {
		return "", "", "", 0, fmt.Errorf("invalid encoded ss payload format")
	}
	cred := decoded[:at]
	hostPort := decoded[at+1:]

	idx := strings.Index(cred, ":")
	if idx < 1 {
		return "", "", "", 0, fmt.Errorf("invalid ss credential section")
	}
	method = strings.TrimSpace(cred[:idx])
	password = strings.TrimSpace(cred[idx+1:])

	h, pstr, err := net.SplitHostPort(strings.TrimSpace(hostPort))
	if err != nil {
		return "", "", "", 0, fmt.Errorf("invalid ss host:port: %w", err)
	}
	p, err := strconv.Atoi(pstr)
	if err != nil || p <= 0 {
		return "", "", "", 0, fmt.Errorf("invalid ss port")
	}

	return method, password, h, p, nil
}

func parseShadowsocksRawURL(u *url.URL) (*ShadowsocksURL, error) {
	if u == nil {
		return nil, fmt.Errorf("empty ss url")
	}

	var (
		method   string
		password string
		host     string
		port     int
		err      error
	)

	if strings.TrimSpace(u.Host) != "" {
		h, pstr, hostErr := net.SplitHostPort(u.Host)
		if hostErr != nil {
			return nil, fmt.Errorf("invalid ss host:port: %w", hostErr)
		}
		host = h
		port, err = strconv.Atoi(pstr)
		if err != nil || port <= 0 {
			return nil, fmt.Errorf("invalid ss port")
		}

		method, password, err = parseSSMethodPassword(u.User)
		if err != nil {
			return nil, err
		}
	} else {
		method, password, host, port, err = parseSSFullEncoded(u.String())
		if err != nil {
			return nil, err
		}
	}

	if method == "" || password == "" || host == "" || port <= 0 {
		return nil, fmt.Errorf("incomplete ss configuration")
	}

	q := u.Query()
	cfg := &ShadowsocksConfig{
		Method:   method,
		Password: password,
		Address:  host,
		Port:     port,
		UoT:      parseFlexibleBool(pickQueryValue(q, "uot"), false),
		Remark:   u.Fragment,
		raw:      u,
	}

	return &ShadowsocksURL{
		Config: cfg,
	}, nil
}

func ParseShadowsocksURL(u *url.URL) (*ShadowsocksURL, error) {
	su, err := parseShadowsocksRawURL(u)
	if err == nil {
		return su, nil
	}

	decoded, derr := decodeProtocolBase64URL(u, "shadowsocks", "ss")
	if derr != nil {
		return nil, err
	}

	// Accept decoded payloads without explicit canonical scheme.
	if decoded.Scheme == "ss" {
		decoded.Scheme = "shadowsocks"
	}

	su, err = parseShadowsocksRawURL(decoded)
	if err != nil {
		return nil, err
	}
	return su, nil
}
