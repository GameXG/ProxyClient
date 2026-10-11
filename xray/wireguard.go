package xray

import (
	"encoding/json"
	"fmt"
	"net/url"
	"runtime"

	core "github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

func WireGuardToXRay(wu *WireGuardURL, port int) ([]byte, int, error) {
	var err error
	if port < 1 {
		port, err = getFreePort()
		if err != nil {
			return nil, 0, err
		}
	}

	cfg := wu.Config
	peer := map[string]interface{}{
		"publicKey": cfg.PublicKey,
		"endpoint":  cfg.Endpoint,
	}
	if len(cfg.AllowedIPs) > 0 {
		peer["allowedIPs"] = cfg.AllowedIPs
	}
	if cfg.KeepAlive > 0 {
		peer["keepAlive"] = cfg.KeepAlive
	}

	wireguardSettings := map[string]interface{}{
		"secretKey":      cfg.SecretKey,
		"address":        cfg.Address,
		"peers":          []map[string]interface{}{peer},
		"domainStrategy": cfg.DomainStrategy,
	}
	if len(cfg.Reserved) == 3 {
		wireguardSettings["reserved"] = cfg.Reserved
	}
	if cfg.MTU > 0 {
		wireguardSettings["mtu"] = cfg.MTU
	}
	if cfg.Workers > 0 {
		wireguardSettings["workers"] = cfg.Workers
	}

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
				Tag:      "wireguard-out",
				Protocol: "wireguard",
				Settings: wireguardSettings,
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
	}

	buf, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal wireguard config to json: %w", err)
	}
	return buf, port, nil
}

func StartWireGuard(u *url.URL, port int) (*core.Instance, int, error) {
	wireguardURL := u.String()
	server := getServer(wireguardURL)
	if server != nil {
		return server.Instance, server.SocksPort, nil
	}

	wu, err := ParseWireGuardURL(u)
	if err != nil {
		return nil, 0, err
	}

	jsonConfig, port, err := WireGuardToXRay(wu, port)
	if err != nil {
		return nil, 0, err
	}

	instance, err := core.StartInstance("json", jsonConfig)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to start wireguard xray instance: %w", err)
	}

	setServer(wireguardURL, instance, port)
	return instance, port, nil
}
