package xray

import (
	"encoding/json"
	"fmt"
	"net/url"
	"runtime"
	"strings"

	core "github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

// TrojanToXRay converts Trojan URL to Xray JSON configuration.
func TrojanToXRay(u *url.URL, port int) ([]byte, int, error) {
	tu, err := ParseTrojanURL(u)
	if err != nil {
		return nil, 0, err
	}
	cfg := tu.Config
	if err := validateTLSVerification(cfg.AllowInsecure, cfg.PinnedPeerCertSha256, cfg.VerifyPeerCertByName); err != nil {
		return nil, 0, err
	}

	if port < 1 {
		port, err = getFreePort()
		if err != nil {
			return nil, 0, err
		}
	}

	trojanSettings := map[string]interface{}{
		"servers": []map[string]interface{}{
			{
				"address":  cfg.Address,
				"port":     cfg.Port,
				"password": cfg.Password,
				"flow":     cfg.Flow,
				"level":    0,
			},
		},
	}

	streamSettings := &StreamSettings{
		Network:  cfg.Type,
		Security: cfg.Security,
	}

	switch cfg.Security {
	case "tls":
		streamSettings.TLSSettings = &TLSSettings{ServerName: cfg.SNI}
		applyTLSVerification(streamSettings.TLSSettings, cfg.PinnedPeerCertSha256, cfg.VerifyPeerCertByName)
		if cfg.Fingerprint != "" {
			streamSettings.TLSSettings.Fingerprint = cfg.Fingerprint
		}
		if cfg.ALPN != "" {
			streamSettings.TLSSettings.ALPN = splitCSV(cfg.ALPN)
		}
	case "xtls":
		streamSettings.Security = "xtls"
		streamSettings.XTLSSettings = &TLSSettings{ServerName: cfg.SNI}
		applyTLSVerification(streamSettings.XTLSSettings, cfg.PinnedPeerCertSha256, cfg.VerifyPeerCertByName)
		if cfg.Fingerprint != "" {
			streamSettings.XTLSSettings.Fingerprint = cfg.Fingerprint
		}
		if cfg.ALPN != "" {
			streamSettings.XTLSSettings.ALPN = splitCSV(cfg.ALPN)
		}
	case "reality":
		streamSettings.Security = "reality"
		streamSettings.RealitySettings = &RealitySettings{
			ServerName:  cfg.SNI,
			Fingerprint: cfg.Fingerprint,
			PublicKey:   cfg.PublicKey,
			ShortID:     cfg.ShortID,
			SpiderX:     cfg.SpiderX,
		}
	}

	switch cfg.Type {
	case "ws":
		streamSettings.WSSettings = &WSSettings{Path: cfg.Path, Host: cfg.Host}
	case "xhttp":
		streamSettings.Network = "xhttp"
		streamSettings.XHTTPSettings = &XHTTPSettings{Host: cfg.Host, Path: cfg.Path, Method: "GET", Version: "h2"}
		if strings.Contains(strings.ToLower(cfg.ALPN), "h3") {
			streamSettings.XHTTPSettings.Version = "h3"
		}
	case "tcp":
		if cfg.Host != "" || cfg.Path != "" {
			streamSettings.TCPSettings = &TCPSettings{Header: &Header{Type: "http", Request: map[string]interface{}{"path": []string{cfg.Path}, "headers": map[string]interface{}{"Host": []string{cfg.Host}}}}}
		}
	case "grpc":
		streamSettings.GRPCSettings = &GRPCSettings{ServiceName: cfg.ServiceName, MultiMode: false}
	case "http", "h2":
		streamSettings.HTTPSettings = &HTTPSettings{Path: cfg.Path}
		if cfg.Host != "" {
			streamSettings.HTTPSettings.Host = []string{cfg.Host}
		}
	}

	config := &XRayConfig{
		Log: &LogConfig{Access: "none", Loglevel: "error"},
		Inbounds: []Inbound{
			buildLocalSocksInbound(port),
		},
		Outbounds: []Outbound{
			{Tag: "trojan-out", Protocol: "trojan", Settings: trojanSettings, StreamSettings: streamSettings, Mux: &Mux{Enabled: false, Concurrency: runtime.NumCPU()}},
			{Tag: "direct", Protocol: "freedom"},
		},
	}

	buf, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal config to json: %w", err)
	}
	return buf, port, nil
}

func StartTrojan(u *url.URL, port int) (*core.Instance, int, error) {
	key := u.String()
	if server := getServer(key); server != nil {
		return server.Instance, server.SocksPort, nil
	}

	jsonConfig, port, err := TrojanToXRay(u, port)
	if err != nil {
		return nil, 0, err
	}
	instance, err := core.StartInstance("json", jsonConfig)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to start xray instance: %w", err)
	}
	setServer(key, instance, port)
	return instance, port, nil
}
