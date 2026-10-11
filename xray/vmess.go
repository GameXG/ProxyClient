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

// VmessToXRay converts VMess URL to Xray JSON configuration
func VmessToXRay(vmess *VmessConfig, port int) ([]byte, int, error) {
	if err := validateTLSVerification(vmess.AllowInsecure, vmess.PinnedPeerCertSha256, vmess.VerifyPeerCertByName); err != nil {
		return nil, 0, err
	}
	var err error
	if port < 1 {
		port, err = getFreePort()
		if err != nil {
			return nil, 0, err
		}
	}

	// If vmess.Net is "xhttp", we should handle it properly
	// This should typically be in the JSON processing part after base64 decoding

	// Generate complete Xray configuration
	config := createCompleteVmessConfig(vmess, port)

	// Return JSON format
	buf, err := json.MarshalIndent(config, "", "  ")
	return buf, port, err
}

func createCompleteVmessConfig(vmess *VmessConfig, port int) *XRayConfig {
	// If it's WebSocket and meets the auto-conversion conditions, prioritize using XHTTP
	if vmess.Net == "ws" && vmess.XHTTPVer != "" {
		vmess.Net = "xhttp"
	}

	return &XRayConfig{
		Log: &LogConfig{
			Access: "none", // Disable access logs
			// Error:    "none", // Disable error log file (Loglevel controls console error log verbosity)
			Loglevel: "error", // Set to "none" to disable console error logs, or "error" for errors only
		},
		Inbounds: []Inbound{
			buildLocalSocksInbound(port),
		},
		Outbounds: []Outbound{
			{
				Tag:      "vmess-out",
				Protocol: "vmess",
				Settings: map[string]interface{}{
					"vnext": []map[string]interface{}{
						{
							"address": vmess.Add,
							"port":    vmess.Port.Value(),
							"users": []map[string]interface{}{
								{
									"id":       vmess.ID,
									"alterId":  vmess.Aid.Value(),
									"security": getSecurityMethod(vmess),
									"level":    0,
									"flow":     vmess.Flow, // XTLS Flow support
								},
							},
						},
					},
				},
				StreamSettings: buildEnhancedStreamSettings(vmess),
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
		// DomainStrategy: "AsIs",
		// Rules: []RoutingRule{
		// 	{
		// 		Type:        "field",
		// 		OutboundTag: "direct",
		// 		IP:          []string{"geoip:private"},
		// 	},
		// },
		// },
	}
}

func getSecurityMethod(vmess *VmessConfig) string {
	if vmess.Security != "" {
		return vmess.Security
	}
	return "auto"
}

// Modified buildEnhancedStreamSettings function
func buildEnhancedStreamSettings(vmess *VmessConfig) *StreamSettings {
	ss := &StreamSettings{
		Network:  vmess.Net,
		Security: vmess.TLS.Value(),
	}

	tls := vmess.TLS.Value()

	// Configure TLS
	if tls == "tls" || tls == "true" {
		ss.TLSSettings = &TLSSettings{
			ServerName: vmess.Host,
		}
		applyTLSVerification(ss.TLSSettings, vmess.PinnedPeerCertSha256, vmess.VerifyPeerCertByName)

		if vmess.SNI != "" {
			ss.TLSSettings.ServerName = vmess.SNI
		}

		if vmess.Alpn != "" {
			ss.TLSSettings.ALPN = strings.Split(vmess.Alpn, ",")
		}

		if vmess.Fp != "" {
			ss.TLSSettings.Fingerprint = vmess.Fp
		}
	}

	// Configure XTLS
	if tls == "xtls" {
		ss.Security = "xtls"
		ss.XTLSSettings = &TLSSettings{
			ServerName: vmess.Host,
		}
		applyTLSVerification(ss.XTLSSettings, vmess.PinnedPeerCertSha256, vmess.VerifyPeerCertByName)

		if vmess.SNI != "" {
			ss.XTLSSettings.ServerName = vmess.SNI
		}

		if vmess.Alpn != "" {
			ss.XTLSSettings.ALPN = strings.Split(vmess.Alpn, ",")
		}

		if vmess.Fp != "" {
			ss.XTLSSettings.Fingerprint = vmess.Fp
		}
	}

	// Configure Reality
	if tls == "reality" {
		ss.Security = "reality"
		ss.RealitySettings = &RealitySettings{
			ServerName:  vmess.SNI,
			Fingerprint: vmess.Fp,
			PublicKey:   vmess.PbK,
			ShortID:     vmess.Sid,
			SpiderX:     vmess.SpX,
		}
	}

	// Configure settings based on network type
	switch vmess.Net {
	case "ws":
		// Retain original WebSocket handling
		configureWS(ss, vmess)

	case "xhttp": // Add support for explicitly using XHTTP
		configureXHTTP(ss, vmess)
	case "kcp":
		configureKCP(ss, vmess)
	case "tcp":
		configureTCP(ss, vmess)
	case "http", "h2":
		configureHTTP(ss, vmess)
	case "quic":
		configureQUIC(ss, vmess)
	case "grpc":
		configureGRPC(ss, vmess)
	}

	return ss
}

// Add XHTTP configuration function
func configureXHTTP(ss *StreamSettings, vmess *VmessConfig) {
	ss.XHTTPSettings = &XHTTPSettings{
		Host:    vmess.Host,
		Path:    vmess.Path,
		Method:  "GET",
		Version: "h2", // Default to HTTP/2
	}

	// Select HTTP version based on possible ALPN settings
	if vmess.Alpn != "" {
		if strings.Contains(vmess.Alpn, "h3") {
			ss.XHTTPSettings.Version = "h3"
		}
	}
}

// Retain the original WebSocket configuration, but modify to use the independent host field
func configureWS(ss *StreamSettings, vmess *VmessConfig) {
	ss.WSSettings = &WSSettings{
		Path: vmess.Path,
		Host: vmess.Host, // Use independent Host field
	}

	// Retain other possible headers, but don't include Host
	if vmess.Host != "" && len(ss.WSSettings.Headers) > 0 {
		delete(ss.WSSettings.Headers, "Host")
		if len(ss.WSSettings.Headers) == 0 {
			ss.WSSettings.Headers = nil
		}
	}
}

func configureTCP(ss *StreamSettings, vmess *VmessConfig) {
	if vmess.Type == "http" {
		ss.TCPSettings = &TCPSettings{
			Header: &Header{
				Type: "http",
				Request: map[string]interface{}{
					"path": []string{vmess.Path},
					"headers": map[string]interface{}{
						"Host": []string{vmess.Host},
					},
				},
			},
		}
	}
}

func configureKCP(ss *StreamSettings, vmess *VmessConfig) {
	ss.KCPSettings = &KCPSettings{
		MTU:              1350,
		TTI:              20,
		UplinkCapacity:   5,
		DownlinkCapacity: 20,
		Congestion:       false,
		ReadBufferSize:   1,
		WriteBufferSize:  1,
	}

	if vmess.Type != "" {
		ss.KCPSettings.Header = &Header{
			Type: vmess.Type,
		}
	}
}

func configureHTTP(ss *StreamSettings, vmess *VmessConfig) {
	ss.HTTPSettings = &HTTPSettings{
		Path: vmess.Path,
	}

	if vmess.Host != "" {
		ss.HTTPSettings.Host = []string{vmess.Host}
	}
}

func configureQUIC(ss *StreamSettings, vmess *VmessConfig) {
	ss.QUICSettings = &QUICSettings{
		Security: "none",
	}

	if vmess.Type != "" {
		ss.QUICSettings.Header = &Header{
			Type: vmess.Type,
		}
	}
}

func configureGRPC(ss *StreamSettings, vmess *VmessConfig) {
	ss.GRPCSettings = &GRPCSettings{
		ServiceName: vmess.Path,
		MultiMode:   true,
	}
}

// StartVmess starts a VMess client
func StartVmess(u *url.URL, port int) (*core.Instance, int, error) {

	vmessURL := u.String()

	// Check if already running
	server := getServer(vmessURL)
	if server != nil {
		return server.Instance, server.SocksPort, nil
	}

	vu, err := ParseVmessURL(u)
	if err != nil {
		return nil, 0, err
	}

	// Get JSON configuration
	jsonConfig, port, err := VmessToXRay(vu.Config, port)
	if err != nil {
		return nil, 0, err
	}

	// Directly use Xray's StartInstance function to create server configuration
	instance, err := core.StartInstance("json", jsonConfig)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to start Xray instance: %w", err)
	}

	setServer(vmessURL, instance, port)

	return instance, port, nil
}
