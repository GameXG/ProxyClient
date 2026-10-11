package xray

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// TrojanConfig stores Trojan URL parameters
// trojan://password@host:port?security=tls&type=tcp&sni=example.com...
type TrojanConfig struct {
	Password             string
	Address              string
	Port                 int
	Flow                 string
	Type                 string
	Security             string
	Path                 string
	Host                 string
	SNI                  string
	ALPN                 string
	Fingerprint          string
	PublicKey            string
	ShortID              string
	SpiderX              string
	ServiceName          string
	AllowInsecure        bool
	PinnedPeerCertSha256 string
	VerifyPeerCertByName string
	Remark               string

	raw *url.URL `json:"-"`
}

type TrojanURL struct {
	Config *TrojanConfig
}

func parseTrojanRawURL(u *url.URL) (*TrojanURL, error) {
	if u == nil {
		return nil, fmt.Errorf("empty trojan url")
	}
	if u.User == nil {
		return nil, fmt.Errorf("missing password in trojan url")
	}
	password := u.User.Username()

	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, fmt.Errorf("invalid host:port in trojan url: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return nil, fmt.Errorf("invalid port in trojan url: %v", portStr)
	}

	cfg := &TrojanConfig{
		Password:      password,
		Address:       host,
		Port:          port,
		Security:      "tls",
		Type:          "tcp",
		AllowInsecure: false,
		Remark:        u.Fragment,
		raw:           u,
	}

	q := u.Query()
	if v := strings.TrimSpace(q.Get("flow")); v != "" {
		cfg.Flow = v
	}
	if v := strings.TrimSpace(q.Get("type")); v != "" {
		cfg.Type = v
	}
	if v := strings.TrimSpace(q.Get("security")); v != "" {
		cfg.Security = v
	}
	if v := strings.TrimSpace(q.Get("path")); v != "" {
		cfg.Path = v
	}
	if v := strings.TrimSpace(q.Get("host")); v != "" {
		cfg.Host = v
	}
	if v := strings.TrimSpace(q.Get("sni")); v != "" {
		cfg.SNI = v
	} else if cfg.Host != "" {
		cfg.SNI = cfg.Host
	} else {
		cfg.SNI = host
	}
	if v := strings.TrimSpace(q.Get("alpn")); v != "" {
		cfg.ALPN = v
	}
	if v := strings.TrimSpace(q.Get("fp")); v != "" {
		cfg.Fingerprint = v
	}
	if v := strings.TrimSpace(q.Get("pbk")); v != "" {
		cfg.PublicKey = v
	}
	if v := strings.TrimSpace(q.Get("sid")); v != "" {
		cfg.ShortID = v
	}
	if v := strings.TrimSpace(q.Get("spx")); v != "" {
		cfg.SpiderX = v
	}
	if v := strings.TrimSpace(q.Get("serviceName")); v != "" {
		cfg.ServiceName = v
	}
	if v := strings.TrimSpace(q.Get("allowInsecure")); v != "" {
		cfg.AllowInsecure = parseFlexibleBool(v, false)
	}
	if v := pickQueryValue(q, "pcs", "pinnedPeerCertSha256"); v != "" {
		cfg.PinnedPeerCertSha256 = v
	}
	if v := pickQueryValue(q, "vcn", "verifyPeerCertByName"); v != "" {
		cfg.VerifyPeerCertByName = v
	}

	return &TrojanURL{Config: cfg}, nil
}

func ParseTrojanURL(u *url.URL) (*TrojanURL, error) {
	tu, err := parseTrojanRawURL(u)
	if err == nil {
		return tu, nil
	}

	decoded, derr := decodeProtocolBase64URL(u, "trojan")
	if derr != nil {
		return nil, err
	}

	tu, err = parseTrojanRawURL(decoded)
	if err != nil {
		return nil, err
	}
	return tu, nil
}
