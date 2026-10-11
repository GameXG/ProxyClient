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

// VlessToXRay converts VLESS URL to Xray JSON configuration
func VlessToXRay(vu *VlessURL, port int) ([]byte, int, error) {
	cfg := vu.Config
	if err := validateTLSVerification(cfg.AllowInsecure, cfg.PinnedPeerCertSha256, cfg.VerifyPeerCertByName); err != nil {
		return nil, 0, err
	}

	var err error
	// Get a free port (if not provided)
	if port < 1 {
		port, err = getFreePort()
		if err != nil {
			return nil, 0, err
		}
	}

	// Create VLESS outbound configuration
	vlessSettings := map[string]interface{}{
		"vnext": []map[string]interface{}{
			{
				"address": cfg.Address,
				"port":    cfg.Port,
				"users": []map[string]interface{}{
					{
						"id":         cfg.UUID,
						"flow":       cfg.Flow,
						"encryption": cfg.Encryption,
						"level":      0,
					},
				},
			},
		},
	}

	// Create stream settings
	streamSettings := &StreamSettings{
		Network:  cfg.Type,
		Security: cfg.Security,
	}

	// Configure TLS - update this section
	if cfg.Security == "tls" {
		streamSettings.TLSSettings = &TLSSettings{
			ServerName: cfg.SNI,
		}
		applyTLSVerification(streamSettings.TLSSettings, cfg.PinnedPeerCertSha256, cfg.VerifyPeerCertByName)

		if cfg.Fingerprint != "" {
			streamSettings.TLSSettings.Fingerprint = cfg.Fingerprint
		}

		if cfg.ALPN != "" {
			streamSettings.TLSSettings.ALPN = strings.Split(cfg.ALPN, ",")
		}
	}

	// Configure XTLS - update this section
	if cfg.Security == "xtls" {
		streamSettings.Security = "xtls"
		streamSettings.XTLSSettings = &TLSSettings{
			ServerName: cfg.SNI,
		}
		applyTLSVerification(streamSettings.XTLSSettings, cfg.PinnedPeerCertSha256, cfg.VerifyPeerCertByName)

		if cfg.Fingerprint != "" {
			streamSettings.XTLSSettings.Fingerprint = cfg.Fingerprint
		}

		if cfg.ALPN != "" {
			streamSettings.XTLSSettings.ALPN = strings.Split(cfg.ALPN, ",")
		}
	}

	// Configure Reality
	if cfg.Security == "reality" {
		streamSettings.Security = "reality"
		streamSettings.RealitySettings = &RealitySettings{
			ServerName:  cfg.SNI,
			Fingerprint: cfg.Fingerprint,
			PublicKey:   cfg.PublicKey,
			ShortID:     cfg.ShortID,
			SpiderX:     cfg.SpiderX,
		}
	}

	// Configure based on transport type
	switch cfg.Type {
	case "ws":
		streamSettings.WSSettings = &WSSettings{
			Path: cfg.Path,
			Host: cfg.Host, // Use independent Host field instead of headers
		}

	case "xhttp": // Add direct support for XHTTP
		streamSettings.Network = "xhttp"
		streamSettings.XHTTPSettings = &XHTTPSettings{
			Host:    cfg.Host,
			Path:    cfg.Path,
			Method:  "GET",
			Version: "h2",
		}

		// Select HTTP version based on ALPN settings
		if cfg.ALPN != "" {
			if strings.Contains(cfg.ALPN, "h3") {
				streamSettings.XHTTPSettings.Version = "h3"
			}
		}
	case "tcp":
		if cfg.Host != "" || cfg.Path != "" {
			streamSettings.TCPSettings = &TCPSettings{
				Header: &Header{
					Type: "http",
					Request: map[string]interface{}{
						"path": []string{cfg.Path},
						"headers": map[string]interface{}{
							"Host": []string{cfg.Host},
						},
					},
				},
			}
		}
	case "grpc":
		streamSettings.GRPCSettings = &GRPCSettings{
			ServiceName: cfg.ServiceName,
			MultiMode:   true,
		}
	case "http":
		streamSettings.HTTPSettings = &HTTPSettings{
			Path: cfg.Path,
		}
		if cfg.Host != "" {
			streamSettings.HTTPSettings.Host = []string{cfg.Host}
		}
	}

	// Create complete configuration
	config := &XRayConfig{
		Log: &LogConfig{
			Access:   "none",
			Loglevel: "error",
		},
		Inbounds: []Inbound{
			buildLocalSocksInbound(port),
		},
		Outbounds: []Outbound{
			{
				Tag:            "vless-out",
				Protocol:       "vless",
				Settings:       vlessSettings,
				StreamSettings: streamSettings,
				Mux: &Mux{
					Enabled:     false,
					Concurrency: runtime.NumCPU(),
				},
			},
			{
				Tag:      "direct",
				Protocol: "freedom",
			},
		},
		// Routing: &RoutingConfig{
		// 	Rules: []RoutingRule{
		// 		{
		// 			Type:        "field",
		// 			OutboundTag: "direct",
		// 			IP:          []string{"geoip:private"},
		// 		},
		// 	},
		// },
	}

	// Convert to JSON
	buf, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	return buf, port, nil
}

// StartVless starts a VLESS client and returns Xray instance and local SOCKS port
func StartVless(u *url.URL, port int) (*core.Instance, int, error) {

	vlessURL := u.String()

	// Check if already running
	server := getServer(vlessURL)
	if server != nil {
		return server.Instance, server.SocksPort, nil
	}

	vu, err := ParseVlessURL(u)
	if err != nil {
		return nil, 0, err
	}

	// Convert to Xray JSON configuration
	jsonConfig, port, err := VlessToXRay(vu, port)
	if err != nil {
		return nil, 0, err
	}

	// Start Xray instance
	instance, err := core.StartInstance("json", jsonConfig)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to start Xray instance: %w", err)
	}

	// Register the running server
	setServer(vlessURL, instance, port)

	// fmt.Printf("VLESS proxy started on socks5://127.0.0.1:%d\n", port)
	return instance, port, nil
}
