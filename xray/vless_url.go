package xray

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// VlessConfig stores VLESS URL parameters
type VlessConfig struct {
	UUID                 string
	Address              string
	Port                 int
	Encryption           string
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
	AllowInsecure        bool // Controls whether to allow insecure TLS connections
	PinnedPeerCertSha256 string
	VerifyPeerCertByName string
	Remark               string

	raw *url.URL `json:"-"`
}

type VlessURL struct {
	Config *VlessConfig
}

func (v *VlessURL) Raw() *url.URL {
	if v.Config == nil {
		return nil
	}
	return v.Config.raw
}

func (v *VlessURL) Opaque() string {
	if v.Config == nil || v.Config.raw == nil {
		return ""
	}
	return strings.TrimPrefix(v.Config.raw.String(), "vless://")
}

func (v *VlessURL) Host() string {
	if v.Config == nil {
		return ""
	}
	return v.Config.Address
}

func (v *VlessURL) Port() string {
	if v.Config == nil {
		return ""
	}
	return strconv.Itoa(v.Config.Port)
}

func (v *VlessURL) User() string {
	if v.Config == nil {
		return ""
	}
	return v.Config.UUID
}

func (v *VlessURL) Password() string {
	return ""
}

func (v *VlessURL) Protocol() string {
	return "vless"
}

func (v *VlessURL) Name() string {
	if v.Config == nil {
		return ""
	}
	return v.Config.Remark
}

// ParseVlessURL parses VLESS URL
// vless://uuid@host:port?encryption=none&type=tcp&security=tls&sni=example.com...
func parseVlessRawURL(u *url.URL) (*VlessURL, error) {
	// Extract user information
	if u.User == nil {
		return nil, fmt.Errorf("missing user info in VLESS URL")
	}
	uuid := u.User.Username()

	// Extract host and port
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, fmt.Errorf("invalid host:port in VLESS URL: %w", err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port in VLESS URL: %w", err)
	}

	// Create configuration
	cfg := &VlessConfig{
		UUID:          uuid,
		Address:       host,
		Port:          port,
		Encryption:    "none", // VLESS default encryption is none
		Type:          "tcp",  // Default transport type
		AllowInsecure: false,
		raw:           u,
		Remark:        u.Fragment,
	}

	// Parse query parameters
	query := u.Query()

	if v := query.Get("encryption"); v != "" {
		cfg.Encryption = v
	}

	if v := query.Get("flow"); v != "" {
		cfg.Flow = v
	}

	if v := query.Get("type"); v != "" {
		cfg.Type = v
		// XHTTP as explicitly supported type, but not auto-converted
	}

	if v := query.Get("security"); v != "" {
		cfg.Security = v
	}

	if v := query.Get("path"); v != "" {
		cfg.Path = v
	}

	if v := query.Get("host"); v != "" {
		cfg.Host = v
	}

	if v := query.Get("sni"); v != "" {
		cfg.SNI = v
	} else if cfg.Host != "" {
		cfg.SNI = cfg.Host
	}

	if v := query.Get("alpn"); v != "" {
		cfg.ALPN = v
	}

	if v := query.Get("fp"); v != "" {
		cfg.Fingerprint = v
	}

	if v := query.Get("pbk"); v != "" {
		cfg.PublicKey = v
	}

	if v := query.Get("sid"); v != "" {
		cfg.ShortID = v
	}

	if v := query.Get("spx"); v != "" {
		cfg.SpiderX = v
	}

	if v := query.Get("serviceName"); v != "" {
		cfg.ServiceName = v
	}

	if v := query.Get("allowInsecure"); v != "" {
		cfg.AllowInsecure = parseFlexibleBool(v, false)
	}
	if v := pickQueryValue(query, "pcs", "pinnedPeerCertSha256"); v != "" {
		cfg.PinnedPeerCertSha256 = v
	}
	if v := pickQueryValue(query, "vcn", "verifyPeerCertByName"); v != "" {
		cfg.VerifyPeerCertByName = v
	}

	return &VlessURL{
		Config: cfg,
	}, nil
}

func ParseVlessURL(u *url.URL) (*VlessURL, error) {
	vu, err := parseVlessRawURL(u)
	if err == nil {
		return vu, nil
	}

	decoded, derr := decodeProtocolBase64URL(u, "vless")
	if derr != nil {
		return nil, err
	}

	vu, err = parseVlessRawURL(decoded)
	if err != nil {
		return nil, err
	}
	return vu, nil
}
