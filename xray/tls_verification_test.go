package xray

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
)

func TestTLSDefaultsDoNotEmitRemovedAllowInsecure(t *testing.T) {
	parsed, err := url.Parse("vmess://00000000-0000-0000-0000-000000000000@example.com:443?net=ws&tls=tls&sni=example.com")
	if err != nil {
		t.Fatal(err)
	}
	vmess, err := ParseVmessURL(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if vmess.Config.AllowInsecure {
		t.Fatal("VMess must verify TLS by default")
	}
	config, _, err := VmessToXRay(vmess.Config, 1080)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(config), "allowInsecure") {
		t.Fatalf("removed allowInsecure field leaked into Xray config: %s", config)
	}
}

func TestOtherTLSProtocolsVerifyByDefault(t *testing.T) {
	tests := []struct {
		name  string
		build func() ([]byte, bool, error)
	}{
		{
			name: "vless",
			build: func() ([]byte, bool, error) {
				u, _ := url.Parse("vless://00000000-0000-0000-0000-000000000000@example.com:443?type=ws&security=tls&sni=example.com")
				parsed, err := ParseVlessURL(u)
				if err != nil {
					return nil, false, err
				}
				config, _, err := VlessToXRay(parsed, 1080)
				return config, parsed.Config.AllowInsecure, err
			},
		},
		{
			name: "trojan",
			build: func() ([]byte, bool, error) {
				u, _ := url.Parse("trojan://password@example.com:443?type=ws&security=tls&sni=example.com")
				parsed, err := ParseTrojanURL(u)
				if err != nil {
					return nil, false, err
				}
				config, _, err := TrojanToXRay(u, 1080)
				return config, parsed.Config.AllowInsecure, err
			},
		},
		{
			name: "hysteria2",
			build: func() ([]byte, bool, error) {
				u, _ := url.Parse("hysteria2://password@example.com:443?sni=example.com")
				parsed, err := ParseHysteriaURL(u)
				if err != nil {
					return nil, false, err
				}
				config, _, err := HysteriaToXRay(parsed.Config, 1080, "hysteria2")
				return config, parsed.Config.AllowInsecure, err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, allowInsecure, err := test.build()
			if err != nil {
				t.Fatal(err)
			}
			if allowInsecure {
				t.Fatal("TLS must verify certificates by default")
			}
			if strings.Contains(string(config), "allowInsecure") {
				t.Fatalf("removed allowInsecure field leaked into Xray config: %s", config)
			}
		})
	}
}

func TestVmessBase64JSONAcceptsLongTLSReplacementNames(t *testing.T) {
	const pin = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	payload := `{"v":"2","add":"example.com","port":"443","id":"00000000-0000-0000-0000-000000000000","net":"ws","tls":"tls","pinnedPeerCertSha256":"` + pin + `","verifyPeerCertByName":"example.com"}`
	u, err := url.Parse("vmess://" + base64.RawURLEncoding.EncodeToString([]byte(payload)))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseVmessURL(u)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Config.PinnedPeerCertSha256 != pin || parsed.Config.VerifyPeerCertByName != "example.com" {
		t.Fatalf("long TLS replacement aliases were not preserved: %+v", parsed.Config)
	}
}

func TestTLSReplacementParametersAreSerialized(t *testing.T) {
	const pin = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	parsed, err := url.Parse("vmess://00000000-0000-0000-0000-000000000000@example.com:443?net=ws&tls=tls&pcs=" + pin + "&vcn=example.com")
	if err != nil {
		t.Fatal(err)
	}
	vmess, err := ParseVmessURL(parsed)
	if err != nil {
		t.Fatal(err)
	}
	config, _, err := VmessToXRay(vmess.Config, 1080)
	if err != nil {
		t.Fatal(err)
	}
	text := string(config)
	if !strings.Contains(text, `"pinnedPeerCertSha256": "`+pin+`"`) {
		t.Fatalf("certificate pin missing from Xray config: %s", config)
	}
	if !strings.Contains(text, `"verifyPeerCertByName": "example.com"`) {
		t.Fatalf("certificate name missing from Xray config: %s", config)
	}
}

func TestLegacyAllowInsecureRequiresSafeReplacement(t *testing.T) {
	parsed, err := url.Parse("vmess://00000000-0000-0000-0000-000000000000@example.com:443?net=ws&tls=tls&allowInsecure=1")
	if err != nil {
		t.Fatal(err)
	}
	vmess, err := ParseVmessURL(parsed)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = VmessToXRay(vmess.Config, 1080)
	if err == nil || !strings.Contains(err.Error(), "pcs/pinnedPeerCertSha256") {
		t.Fatalf("expected actionable allowInsecure migration error, got %v", err)
	}
}
