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

func buildHysteriaStreamSettings(cfg *HysteriaConfig, protocol string) *StreamSettings {
	network := protocol
	if network != "hysteria2" {
		network = "hysteria"
	}

	ss := &StreamSettings{
		Network:  network,
		Security: "tls",
		TLSSettings: &TLSSettings{
			ServerName: cfg.SNI,
		},
	}
	applyTLSVerification(ss.TLSSettings, cfg.PinnedPeerCertSha256, cfg.VerifyPeerCertByName)
	if len(cfg.ALPN) > 0 {
		ss.TLSSettings.ALPN = cfg.ALPN
	}
	if cfg.Fingerprint != "" {
		ss.TLSSettings.Fingerprint = cfg.Fingerprint
	}

	if cfg.Auth != "" || cfg.Congestion != "" || cfg.UpMbps > 0 || cfg.DownMbps > 0 || cfg.Obfs != "" || cfg.ObfsPassword != "" {
		hs := &HysteriaSettings{
			Version:    2,
			Auth:       cfg.Auth,
			Congestion: cfg.Congestion,
		}
		if cfg.UpMbps > 0 {
			hs.Up = fmt.Sprintf("%d Mbps", cfg.UpMbps)
		}
		if cfg.DownMbps > 0 {
			hs.Down = fmt.Sprintf("%d Mbps", cfg.DownMbps)
		}
		if cfg.Obfs != "" {
			hs.Obfs = cfg.Obfs
		}
		if cfg.ObfsPassword != "" {
			hs.Password = cfg.ObfsPassword
		}
		ss.HysteriaSettings = hs
	}

	return ss
}

func hysteriaOutboundSettings(cfg *HysteriaConfig, protocol string) map[string]interface{} {
	if protocol == "hysteria2" {
		settings := map[string]interface{}{
			"server":            cfg.Address,
			"server_port":       cfg.Port,
			"use_udp_extension": cfg.UseUDPExtension,
		}
		if cfg.Password != "" {
			settings["password"] = cfg.Password
		}
		if cfg.HopPorts != "" {
			settings["hopPorts"] = cfg.HopPorts
		}
		return settings
	}

	settings := map[string]interface{}{
		"server":      cfg.Address,
		"server_port": cfg.Port,
		"up_mbps":     cfg.UpMbps,
		"down_mbps":   cfg.DownMbps,
		"protocol":    cfg.Protocol,
		"congestion":  cfg.Congestion,
	}
	if cfg.Auth != "" {
		settings["auth_str"] = cfg.Auth
	} else if cfg.Password != "" {
		settings["auth_str"] = cfg.Password
	}
	if cfg.Obfs != "" {
		settings["obfs"] = cfg.Obfs
	}
	if cfg.ObfsPassword != "" {
		settings["obfs_param"] = cfg.ObfsPassword
	}
	return settings
}

func HysteriaToXRay(cfg *HysteriaConfig, port int, protocol string) ([]byte, int, error) {
	if err := validateTLSVerification(cfg.AllowInsecure, cfg.PinnedPeerCertSha256, cfg.VerifyPeerCertByName); err != nil {
		return nil, 0, err
	}
	var err error
	if port < 1 {
		port, err = getFreePort()
		if err != nil {
			return nil, 0, err
		}
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
				Tag:            "hysteria-out",
				Protocol:       protocol,
				Settings:       hysteriaOutboundSettings(cfg, protocol),
				StreamSettings: buildHysteriaStreamSettings(cfg, protocol),
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
		return nil, 0, fmt.Errorf("failed to marshal hysteria config to json: %w", err)
	}
	return buf, port, nil
}

func StartHysteria(u *url.URL, port int) (*core.Instance, int, error) {
	hysteriaURL := u.String()
	if server := getServer(hysteriaURL); server != nil {
		return server.Instance, server.SocksPort, nil
	}

	hu, err := ParseHysteriaURL(u)
	if err != nil {
		return nil, 0, err
	}

	if port < 1 {
		port, err = getFreePort()
		if err != nil {
			return nil, 0, err
		}
	}

	primary := "hysteria2"
	if hu.Config.Scheme == "hysteria" {
		primary = "hysteria"
	}
	fallback := "hysteria"
	if primary == "hysteria" {
		fallback = "hysteria2"
	}

	candidates := []string{primary}
	if fallback != primary {
		candidates = append(candidates, fallback)
	}

	var errs []string
	for _, protocol := range candidates {
		jsonConfig, _, confErr := HysteriaToXRay(hu.Config, port, protocol)
		if confErr != nil {
			errs = append(errs, fmt.Sprintf("%s config error: %v", protocol, confErr))
			continue
		}

		instance, startErr := core.StartInstance("json", jsonConfig)
		if startErr != nil {
			errs = append(errs, fmt.Sprintf("%s start error: %v", protocol, startErr))
			continue
		}

		setServer(hysteriaURL, instance, port)
		return instance, port, nil
	}

	return nil, 0, fmt.Errorf("failed to start hysteria instance: %s", strings.Join(errs, "; "))
}
