package xray

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// HysteriaConfig stores Hysteria/Hysteria2 URL parameters.
// Supported schemes: hysteria://, hysteria2://, hy2://
type HysteriaConfig struct {
	Scheme               string
	Address              string
	Port                 int
	Password             string
	Auth                 string
	HopPorts             string
	UseUDPExtension      bool
	Protocol             string
	UpMbps               int
	DownMbps             int
	Congestion           string
	Obfs                 string
	ObfsPassword         string
	SNI                  string
	ALPN                 []string
	Fingerprint          string
	AllowInsecure        bool
	PinnedPeerCertSha256 string
	VerifyPeerCertByName string
	Remark               string

	raw *url.URL `json:"-"`
}

type HysteriaURL struct {
	Config *HysteriaConfig
}

func parseHysteriaRawURL(u *url.URL) (*HysteriaURL, error) {
	if u == nil {
		return nil, fmt.Errorf("empty hysteria url")
	}

	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return nil, fmt.Errorf("missing host in hysteria url")
	}

	port := parseFlexibleInt(strings.TrimSpace(u.Port()), 443)
	if port <= 0 {
		return nil, fmt.Errorf("invalid hysteria port")
	}

	q := u.Query()
	password := pickQueryValue(q, "password")
	auth := pickQueryValue(q, "auth", "auth_str", "token")

	if u.User != nil {
		username := strings.TrimSpace(u.User.Username())
		if rawPassword, hasPassword := u.User.Password(); hasPassword {
			rawPassword = strings.TrimSpace(rawPassword)
			if password == "" && rawPassword != "" {
				password = rawPassword
			}
			if auth == "" && rawPassword != "" {
				auth = rawPassword
			}
			if password == "" && username != "" {
				password = username
			}
			if auth == "" && username != "" {
				auth = username
			}
		} else if username != "" {
			if password == "" {
				password = username
			}
			if auth == "" {
				auth = username
			}
		}
	}

	if password == "" && auth != "" {
		password = auth
	}
	if auth == "" && password != "" {
		auth = password
	}

	cfg := &HysteriaConfig{
		Scheme:          strings.ToLower(strings.TrimSpace(u.Scheme)),
		Address:         host,
		Port:            port,
		Password:        password,
		Auth:            auth,
		HopPorts:        pickQueryValue(q, "hopPorts", "hop_ports"),
		UseUDPExtension: parseFlexibleBool(pickQueryValue(q, "useUdpExtension", "use_udp_extension"), true),
		Protocol:        pickQueryValue(q, "protocol"),
		UpMbps:          parseFlexibleInt(pickQueryValue(q, "up_mbps", "upMbps", "up"), 100),
		DownMbps:        parseFlexibleInt(pickQueryValue(q, "down_mbps", "downMbps", "down"), 100),
		Congestion:      pickQueryValue(q, "congestion"),
		Obfs:            pickQueryValue(q, "obfs"),
		ObfsPassword:    pickQueryValue(q, "obfsParam", "obfs_param", "obfsPassword", "obfs_password"),
		SNI:             pickQueryValue(q, "sni"),
		ALPN:            splitCSV(pickQueryValue(q, "alpn")),
		Fingerprint:     pickQueryValue(q, "fp", "fingerprint"),
		AllowInsecure: parseFlexibleBool(
			pickQueryValue(q, "allowInsecure", "insecure", "skip-cert-verify", "allow_insecure"),
			false,
		),
		PinnedPeerCertSha256: pickQueryValue(q, "pcs", "pinnedPeerCertSha256"),
		VerifyPeerCertByName: pickQueryValue(q, "vcn", "verifyPeerCertByName"),
		Remark:               u.Fragment,
		raw:                  u,
	}

	if cfg.Protocol == "" {
		cfg.Protocol = "udp"
	}
	if cfg.Congestion == "" {
		cfg.Congestion = "bbr"
	}
	if cfg.SNI == "" {
		cfg.SNI = host
	}

	return &HysteriaURL{Config: cfg}, nil
}

func ParseHysteriaURL(u *url.URL) (*HysteriaURL, error) {
	hu, err := parseHysteriaRawURL(u)
	if err == nil {
		return hu, nil
	}

	decoded, derr := decodeProtocolBase64URL(u, "hysteria2", "hysteria", "hy2")
	if derr != nil {
		return nil, err
	}
	if decoded.Scheme == "" {
		decoded.Scheme = "hysteria2"
	}

	hu, err = parseHysteriaRawURL(decoded)
	if err != nil {
		return nil, err
	}
	return hu, nil
}

func (h *HysteriaURL) String() string {
	if h == nil || h.Config == nil || h.Config.raw == nil {
		return ""
	}
	return h.Config.raw.String()
}

func (h *HysteriaURL) PortString() string {
	if h == nil || h.Config == nil {
		return ""
	}
	return strconv.Itoa(h.Config.Port)
}
