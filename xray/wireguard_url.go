package xray

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// WireGuardConfig stores WireGuard URL parameters.
// Example:
// wireguard://secretKey@host:port?publicKey=xxx&address=10.0.0.2/32&allowedIPs=0.0.0.0/0,::/0
type WireGuardConfig struct {
	SecretKey      string
	PublicKey      string
	Endpoint       string
	Address        []string
	AllowedIPs     []string
	Reserved       []int
	KeepAlive      int
	MTU            int
	Workers        int
	DomainStrategy string
	Remark         string

	raw *url.URL `json:"-"`
}

type WireGuardURL struct {
	Config *WireGuardConfig
}

func parseWireGuardRawURL(u *url.URL) (*WireGuardURL, error) {
	if u == nil {
		return nil, fmt.Errorf("empty wireguard url")
	}

	q := u.Query()
	secretKey := pickQueryValue(q, "secretKey", "secret_key", "privateKey", "private_key")
	if secretKey == "" && u.User != nil {
		secretKey = strings.TrimSpace(u.User.Username())
	}
	if secretKey == "" {
		return nil, fmt.Errorf("missing secretKey in wireguard url")
	}

	publicKey := pickQueryValue(q, "publicKey", "public_key", "peerPublicKey", "peer_public_key")
	if publicKey == "" {
		return nil, fmt.Errorf("missing publicKey in wireguard url")
	}

	endpoint := pickQueryValue(q, "endpoint")
	if endpoint == "" {
		host := strings.TrimSpace(u.Hostname())
		if host == "" {
			return nil, fmt.Errorf("missing host in wireguard url")
		}
		port := parseFlexibleInt(u.Port(), 0)
		if port <= 0 {
			port = parseFlexibleInt(pickQueryValue(q, "port"), 2408)
		}
		endpoint = net.JoinHostPort(host, strconv.Itoa(port))
	} else {
		endpoint = strings.TrimSpace(endpoint)
		if _, _, err := net.SplitHostPort(endpoint); err != nil {
			return nil, fmt.Errorf("invalid wireguard endpoint %q: %w", endpoint, err)
		}
	}

	addresses := parseAddressList(pickQueryValue(q, "address", "addresses", "localAddress", "local_address"), nil)
	if len(addresses) == 0 {
		return nil, fmt.Errorf("missing address in wireguard url")
	}

	allowedIPs := parseAddressList(pickQueryValue(q, "allowedIPs", "allowed_ips"), []string{"0.0.0.0/0", "::/0"})

	var reserved []int
	if rawReserved := pickQueryValue(q, "reserved"); rawReserved != "" {
		items, err := parseIntList(rawReserved)
		if err != nil {
			return nil, fmt.Errorf("invalid reserved in wireguard url: %w", err)
		}
		if len(items) != 3 {
			return nil, fmt.Errorf("wireguard reserved must contain exactly 3 integers")
		}
		reserved = items
	}

	cfg := &WireGuardConfig{
		SecretKey:      secretKey,
		PublicKey:      publicKey,
		Endpoint:       endpoint,
		Address:        addresses,
		AllowedIPs:     allowedIPs,
		Reserved:       reserved,
		KeepAlive:      parseFlexibleInt(pickQueryValue(q, "keepAlive", "keepalive"), 0),
		MTU:            parseFlexibleInt(pickQueryValue(q, "mtu"), 0),
		Workers:        parseFlexibleInt(pickQueryValue(q, "workers"), 0),
		DomainStrategy: pickQueryValue(q, "domainStrategy", "domain_strategy"),
		Remark:         u.Fragment,
		raw:            u,
	}

	if cfg.DomainStrategy == "" {
		cfg.DomainStrategy = "ForceIP"
	}

	return &WireGuardURL{
		Config: cfg,
	}, nil
}

func ParseWireGuardURL(u *url.URL) (*WireGuardURL, error) {
	wu, err := parseWireGuardRawURL(u)
	if err == nil {
		return wu, nil
	}

	decoded, derr := decodeProtocolBase64URL(u, "wireguard")
	if derr != nil {
		return nil, err
	}

	wu, err = parseWireGuardRawURL(decoded)
	if err != nil {
		return nil, err
	}
	return wu, nil
}
