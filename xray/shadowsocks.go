package xray

import (
	"encoding/json"
	"fmt"
	"net/url"
	"runtime"

	core "github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

func ShadowsocksToXRay(su *ShadowsocksURL, port int) ([]byte, int, error) {
	var err error
	if port < 1 {
		port, err = getFreePort()
		if err != nil {
			return nil, 0, err
		}
	}

	cfg := su.Config
	server := map[string]interface{}{
		"address":  cfg.Address,
		"port":     cfg.Port,
		"method":   cfg.Method,
		"password": cfg.Password,
		"level":    0,
	}
	if cfg.UoT {
		server["uot"] = true
	}

	ssSettings := map[string]interface{}{
		"servers": []map[string]interface{}{server},
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
				Tag:      "shadowsocks-out",
				Protocol: "shadowsocks",
				Settings: ssSettings,
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
		return nil, 0, fmt.Errorf("failed to marshal shadowsocks config to json: %w", err)
	}
	return buf, port, nil
}

func StartShadowsocks(u *url.URL, port int) (*core.Instance, int, error) {
	ssURL := u.String()
	if server := getServer(ssURL); server != nil {
		return server.Instance, server.SocksPort, nil
	}

	su, err := ParseShadowsocksURL(u)
	if err != nil {
		return nil, 0, err
	}

	jsonConfig, port, err := ShadowsocksToXRay(su, port)
	if err != nil {
		return nil, 0, err
	}

	instance, err := core.StartInstance("json", jsonConfig)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to start shadowsocks xray instance: %w", err)
	}

	setServer(ssURL, instance, port)
	return instance, port, nil
}
