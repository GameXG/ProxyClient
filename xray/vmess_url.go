package xray

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// VmessConfig stores VMess URL parameters
type VmessConfig struct {
	V                    FlexInt    `json:"v"`
	PS                   string     `json:"ps"`               // Remarks
	Add                  string     `json:"add"`              // Address
	Port                 FlexInt    `json:"port"`             // Port
	ID                   string     `json:"id"`               // UUID
	Aid                  FlexInt    `json:"aid"`              // AlterID
	Net                  string     `json:"net"`              // Transport protocol
	Type                 string     `json:"type"`             // Camouflage type
	Host                 string     `json:"host"`             // Camouflage domain
	Path                 string     `json:"path"`             // WebSocket path
	TLS                  FlexString `json:"tls"`              // TLS
	SNI                  string     `json:"sni"`              // TLS SNI
	Alpn                 string     `json:"alpn"`             // ALPN
	Flow                 string     `json:"flow"`             // XTLS Flow
	Fp                   string     `json:"fp"`               // Fingerprint
	PbK                  string     `json:"pbk"`              // PublicKey (Reality)
	Sid                  string     `json:"sid"`              // ShortID (Reality)
	SpX                  string     `json:"spx"`              // SpiderX (Reality)
	Security             string     `json:"security"`         // Encryption method
	XHTTPVer             string     `json:"xver"`             // XHTTP version, "h2" or "h3"
	AllowInsecure        bool       `json:"skip_cert_verify"` // Controls whether to allow insecure TLS connections
	PinnedPeerCertSha256 string     `json:"pcs"`              // SHA-256 pin for the peer certificate
	VerifyPeerCertByName string     `json:"vcn"`              // Certificate name accepted by Xray

	raw *url.URL `json:"-"`
}

type VmessURL struct {
	Config *VmessConfig
}

func (v *VmessURL) Raw() *url.URL {
	if v.Config == nil {
		return nil
	}
	return v.Config.raw
}

func (v *VmessURL) Opaque() string {
	if v.Config == nil || v.Config.raw == nil {
		return ""
	}
	return strings.TrimPrefix(v.Config.raw.String(), "vmess://")
}

func (v *VmessURL) Host() string {
	if v.Config == nil {
		return ""
	}
	return v.Config.Add
}

func (v *VmessURL) Port() string {
	if v.Config == nil {
		return ""
	}
	return strconv.Itoa(v.Config.Port.Value())
}

func (v *VmessURL) Protocol() string {
	return "vmess"
}

func (v *VmessURL) User() string {
	if v.Config == nil {
		return ""
	}
	return v.Config.ID
}

func (v *VmessURL) Password() string {
	return "" // VMess does not use password in the traditional sense
}

func (v *VmessURL) Name() string {
	if v.Config == nil {
		return ""
	}
	return v.Config.PS
}

func parseVmessRawURL(u *url.URL) (*VmessURL, error) {
	if u == nil {
		return nil, fmt.Errorf("empty vmess url")
	}
	if u.User == nil {
		return nil, fmt.Errorf("missing user info in vmess url")
	}
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, fmt.Errorf("invalid host:port in vmess url: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return nil, fmt.Errorf("invalid port in vmess url: %w", err)
	}

	vmess := &VmessConfig{
		V:             2,
		PS:            u.Fragment,
		Add:           host,
		Port:          FlexInt(port),
		ID:            u.User.Username(),
		Aid:           0,
		Net:           "tcp",
		Type:          "none",
		TLS:           "",
		AllowInsecure: false,
	}
	q := u.Query()
	if v := strings.TrimSpace(q.Get("ps")); v != "" {
		vmess.PS = v
	}
	if v := strings.TrimSpace(q.Get("aid")); v != "" {
		vmess.Aid = FlexInt(parseFlexibleInt(v, 0))
	}
	if v := strings.TrimSpace(q.Get("net")); v != "" {
		vmess.Net = v
	}
	if v := strings.TrimSpace(q.Get("type")); v != "" {
		vmess.Type = v
	}
	if v := strings.TrimSpace(q.Get("host")); v != "" {
		vmess.Host = v
	}
	if v := strings.TrimSpace(q.Get("path")); v != "" {
		vmess.Path = v
	}
	if v := strings.TrimSpace(q.Get("tls")); v != "" {
		vmess.TLS = FlexString(v)
	}
	if v := strings.TrimSpace(q.Get("security")); v != "" {
		vmess.Security = v
	}
	if v := strings.TrimSpace(q.Get("encryption")); v != "" && vmess.Security == "" {
		vmess.Security = v
	}
	if v := strings.TrimSpace(q.Get("sni")); v != "" {
		vmess.SNI = v
	}
	if v := strings.TrimSpace(q.Get("alpn")); v != "" {
		vmess.Alpn = v
	}
	if v := strings.TrimSpace(q.Get("flow")); v != "" {
		vmess.Flow = v
	}
	if v := strings.TrimSpace(q.Get("fp")); v != "" {
		vmess.Fp = v
	}
	if v := strings.TrimSpace(q.Get("pbk")); v != "" {
		vmess.PbK = v
	}
	if v := strings.TrimSpace(q.Get("sid")); v != "" {
		vmess.Sid = v
	}
	if v := strings.TrimSpace(q.Get("spx")); v != "" {
		vmess.SpX = v
	}
	if v := strings.TrimSpace(q.Get("xver")); v != "" {
		vmess.XHTTPVer = v
	}
	if v := pickQueryValue(q, "allowInsecure", "insecure", "skip-cert-verify", "skip_cert_verify"); v != "" {
		vmess.AllowInsecure = parseFlexibleBool(v, false)
	}
	if v := pickQueryValue(q, "pcs", "pinnedPeerCertSha256"); v != "" {
		vmess.PinnedPeerCertSha256 = v
	}
	if v := pickQueryValue(q, "vcn", "verifyPeerCertByName"); v != "" {
		vmess.VerifyPeerCertByName = v
	}
	if vmess.SNI == "" && vmess.Host != "" {
		vmess.SNI = vmess.Host
	}
	vmess.raw = u

	return &VmessURL{
		Config: vmess,
	}, nil
}

func parseVmessBase64Payload(encoded string) (*VmessURL, error) {
	decodedText, err := decodeBase64Any(encoded)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}
	decoded := []byte(decodedText)

	// Try VMess JSON first.
	vmess := &VmessConfig{AllowInsecure: false}
	if err := json.Unmarshal(decoded, vmess); err == nil {
		var aliases struct {
			PinnedPeerCertSha256 string `json:"pinnedPeerCertSha256"`
			VerifyPeerCertByName string `json:"verifyPeerCertByName"`
			AllowInsecure        *bool  `json:"allowInsecure"`
			Insecure             *bool  `json:"insecure"`
		}
		if json.Unmarshal(decoded, &aliases) == nil {
			if vmess.PinnedPeerCertSha256 == "" {
				vmess.PinnedPeerCertSha256 = aliases.PinnedPeerCertSha256
			}
			if vmess.VerifyPeerCertByName == "" {
				vmess.VerifyPeerCertByName = aliases.VerifyPeerCertByName
			}
			if aliases.AllowInsecure != nil {
				vmess.AllowInsecure = *aliases.AllowInsecure
			} else if aliases.Insecure != nil {
				vmess.AllowInsecure = *aliases.Insecure
			}
		}
		return &VmessURL{
			Config: vmess,
		}, nil
	}

	// Fallback: base64(raw-url) style.
	decodedText = strings.TrimSpace(string(decoded))
	if decodedText == "" {
		return nil, fmt.Errorf("empty vmess payload")
	}
	if !strings.Contains(decodedText, "://") {
		decodedText = "vmess://" + decodedText
	}
	parsed, err := url.Parse(decodedText)
	if err != nil {
		return nil, fmt.Errorf("decoded vmess url parse failed: %w", err)
	}
	return parseVmessRawURL(parsed)
}

func ParseVmessURL(u *url.URL) (*VmessURL, error) {
	// 1) Try raw URL first.
	if vu, err := parseVmessRawURL(u); err == nil {
		return vu, nil
	}

	// 2) Try base64 payload after vmess://
	vmessURL := strings.TrimSpace(u.String())
	encoded := strings.TrimPrefix(vmessURL, "vmess://")
	if vu, err := parseVmessBase64Payload(encoded); err == nil {
		if vu.Config != nil {
			vu.Config.raw = u
		}
		return vu, nil
	}

	// 3) Try base64(url) style explicitly.
	decodedURL, err := decodeProtocolBase64URL(u, "vmess")
	if err == nil {
		if vu, rerr := parseVmessRawURL(decodedURL); rerr == nil {
			if vu.Config != nil {
				vu.Config.raw = u
			}
			return vu, nil
		}
	}

	return nil, fmt.Errorf("invalid vmess url format")
}
