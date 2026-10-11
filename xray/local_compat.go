package xray

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type FlexInt int

func (i FlexInt) Value() int {
	return int(i)
}

func (i *FlexInt) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "" || text == "null" {
		*i = 0
		return nil
	}

	if text[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*i = 0
			return nil
		}
		v, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("invalid int string %q: %w", s, err)
		}
		*i = FlexInt(v)
		return nil
	}

	var v int
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*i = FlexInt(v)
	return nil
}

type FlexString string

func (s FlexString) Value() string {
	return string(s)
}

func (s *FlexString) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "" || text == "null" {
		*s = ""
		return nil
	}

	if text[0] == '"' {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*s = FlexString(v)
		return nil
	}

	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		if b {
			*s = "true"
		} else {
			*s = "false"
		}
		return nil
	}

	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*s = FlexString(n.String())
		return nil
	}

	return fmt.Errorf("unsupported string json value: %s", text)
}

func getFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected addr type %T", ln.Addr())
	}
	return addr.Port, nil
}

func parseFlexibleBool(raw string, defaultValue bool) bool {
	value := strings.TrimSpace(strings.ToLower(raw))
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func parseFlexibleInt(raw string, defaultValue int) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return n
}

func parseIntList(raw string) ([]int, error) {
	items := splitCSV(raw)
	out := make([]int, 0, len(items))
	for _, item := range items {
		n, err := strconv.Atoi(item)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q: %w", item, err)
		}
		out = append(out, n)
	}
	return out, nil
}

func parseAddressList(raw string, defaults []string) []string {
	values := splitCSV(raw)
	if len(values) == 0 {
		return append([]string(nil), defaults...)
	}
	return values
}

func pickQueryValue(q url.Values, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(q.Get(key)); v != "" {
			return v
		}
	}
	return ""
}

func sortedKeys(values url.Values) []string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func buildLocalSocksInbound(port int) Inbound {
	return Inbound{
		Tag:      "socks-in",
		Port:     port,
		Listen:   "127.0.0.1",
		Protocol: "socks",
		Settings: &SocksSetting{
			Auth: "noauth",
			UDP:  true,
			IP:   "127.0.0.1",
		},
		Sniffing: &Sniffing{
			Enabled:      true,
			DestOverride: []string{"http", "tls"},
		},
	}
}

func decodeBase64Any(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty base64 payload")
	}

	if v, err := base64.RawURLEncoding.DecodeString(raw); err == nil {
		return string(v), nil
	}
	if v, err := base64.URLEncoding.DecodeString(raw); err == nil {
		return string(v), nil
	}
	if v, err := base64.RawStdEncoding.DecodeString(raw); err == nil {
		return string(v), nil
	}
	v, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

func decodeProtocolBase64URL(u *url.URL, canonicalScheme string, aliasSchemes ...string) (*url.URL, error) {
	if u == nil {
		return nil, fmt.Errorf("empty url")
	}

	raw := strings.TrimSpace(u.String())
	if raw == "" {
		return nil, fmt.Errorf("empty url")
	}

	schemes := make([]string, 0, 1+len(aliasSchemes))
	schemes = append(schemes, strings.ToLower(strings.TrimSpace(canonicalScheme)))
	for _, s := range aliasSchemes {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" || s == canonicalScheme {
			continue
		}
		schemes = append(schemes, s)
	}

	var payload string
	lowerRaw := strings.ToLower(raw)
	for _, scheme := range schemes {
		prefix := scheme + "://"
		if strings.HasPrefix(lowerRaw, prefix) {
			payload = raw[len(prefix):]
			break
		}
	}
	if payload == "" {
		return nil, fmt.Errorf("url does not contain expected scheme prefix")
	}

	decoded, err := decodeBase64Any(payload)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" {
		return nil, fmt.Errorf("decoded base64 payload is empty")
	}

	var parsed *url.URL
	if strings.Contains(decoded, "://") {
		parsed, err = url.Parse(decoded)
	} else {
		parsed, err = url.Parse(canonicalScheme + "://" + decoded)
	}
	if err != nil {
		return nil, fmt.Errorf("decoded url parse failed: %w", err)
	}

	return parsed, nil
}
