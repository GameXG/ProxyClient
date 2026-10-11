package xray

// General configuration structure, matching the Xray configuration format
type XRayConfig struct {
	Log       *LogConfig     `json:"log,omitempty"`
	DNS       *DNSConfig     `json:"dns,omitempty"`
	Routing   *RoutingConfig `json:"routing,omitempty"`
	Inbounds  []Inbound      `json:"inbounds"`
	Outbounds []Outbound     `json:"outbounds"`
}

type LogConfig struct {
	Access   string `json:"access,omitempty"`
	Error    string `json:"error,omitempty"`
	Loglevel string `json:"loglevel,omitempty"`
}

type DNSConfig struct {
	Servers []string `json:"servers,omitempty"`
}

type RoutingConfig struct {
	DomainStrategy string        `json:"domainStrategy,omitempty"`
	Rules          []RoutingRule `json:"rules,omitempty"`
}

type RoutingRule struct {
	Type        string   `json:"type"`
	OutboundTag string   `json:"outboundTag"`
	Domain      []string `json:"domain,omitempty"`
	IP          []string `json:"ip,omitempty"`
}

type Inbound struct {
	Tag      string      `json:"tag"`
	Port     int         `json:"port"`
	Listen   string      `json:"listen,omitempty"`
	Protocol string      `json:"protocol"`
	Settings interface{} `json:"settings"`
	Sniffing *Sniffing   `json:"sniffing,omitempty"`
}

type Outbound struct {
	Tag            string          `json:"tag,omitempty"`
	Protocol       string          `json:"protocol"`
	Settings       interface{}     `json:"settings"`
	StreamSettings *StreamSettings `json:"streamSettings,omitempty"`
	Mux            *Mux            `json:"mux,omitempty"`
}

type Sniffing struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
}

type SocksSetting struct {
	Auth      string `json:"auth,omitempty"`
	UDP       bool   `json:"udp,omitempty"`
	IP        string `json:"ip,omitempty"`
	UserLevel int    `json:"userLevel,omitempty"`
}

type StreamSettings struct {
	Network          string            `json:"network,omitempty"`
	Security         string            `json:"security,omitempty"`
	TLSSettings      *TLSSettings      `json:"tlsSettings,omitempty"`
	TCPSettings      *TCPSettings      `json:"tcpSettings,omitempty"`
	KCPSettings      *KCPSettings      `json:"kcpSettings,omitempty"`
	WSSettings       *WSSettings       `json:"wsSettings,omitempty"`
	HTTPSettings     *HTTPSettings     `json:"httpSettings,omitempty"`
	QUICSettings     *QUICSettings     `json:"quicSettings,omitempty"`
	GRPCSettings     *GRPCSettings     `json:"grpcSettings,omitempty"`
	XTLSSettings     *TLSSettings      `json:"xtlsSettings,omitempty"`    // Added XTLS support
	RealitySettings  *RealitySettings  `json:"realitySettings,omitempty"` // Added Reality support
	XHTTPSettings    *XHTTPSettings    `json:"xhttpSettings,omitempty"`   // Added XHTTP support
	HysteriaSettings *HysteriaSettings `json:"hysteriaSettings,omitempty"`
}

type RealitySettings struct {
	Show        bool   `json:"show,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	ServerName  string `json:"serverName,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	ShortID     string `json:"shortId,omitempty"`
	SpiderX     string `json:"spiderX,omitempty"`
}

type TLSSettings struct {
	ServerName           string   `json:"serverName,omitempty"`
	ALPN                 []string `json:"alpn,omitempty"`
	Fingerprint          string   `json:"fingerprint,omitempty"`
	PinnedPeerCertSha256 string   `json:"pinnedPeerCertSha256,omitempty"`
	VerifyPeerCertByName string   `json:"verifyPeerCertByName,omitempty"`
}

type TCPSettings struct {
	Header *Header `json:"header,omitempty"`
}

type Header struct {
	Type     string      `json:"type,omitempty"`
	Request  interface{} `json:"request,omitempty"`
	Response interface{} `json:"response,omitempty"`
}

type KCPSettings struct {
	MTU              int     `json:"mtu,omitempty"`
	TTI              int     `json:"tti,omitempty"`
	UplinkCapacity   int     `json:"uplinkCapacity,omitempty"`
	DownlinkCapacity int     `json:"downlinkCapacity,omitempty"`
	Congestion       bool    `json:"congestion,omitempty"`
	ReadBufferSize   int     `json:"readBufferSize,omitempty"`
	WriteBufferSize  int     `json:"writeBufferSize,omitempty"`
	Header           *Header `json:"header,omitempty"`
}

// Updated WSSettings structure
type WSSettings struct {
	Path    string            `json:"path,omitempty"`
	Host    string            `json:"host,omitempty"` // Added independent Host field
	Headers map[string]string `json:"headers,omitempty"`
}

type HTTPSettings struct {
	Path string   `json:"path,omitempty"`
	Host []string `json:"host,omitempty"`
}

type QUICSettings struct {
	Security string  `json:"security,omitempty"`
	Key      string  `json:"key,omitempty"`
	Header   *Header `json:"header,omitempty"`
}

type GRPCSettings struct {
	ServiceName string `json:"serviceName,omitempty"`
	MultiMode   bool   `json:"multiMode,omitempty"`
}

type Mux struct {
	Enabled     bool   `json:"enabled"`
	Concurrency int    `json:"concurrency,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
}

// Added new XHTTP settings structure
type XHTTPSettings struct {
	Host    string            `json:"host,omitempty"`
	Path    string            `json:"path,omitempty"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Version string            `json:"version,omitempty"` // "h2" or "h3"
}

type HysteriaSettings struct {
	Version             int32  `json:"version,omitempty"`
	Auth                string `json:"auth,omitempty"`
	Up                  string `json:"up,omitempty"`
	Down                string `json:"down,omitempty"`
	Congestion          string `json:"congestion,omitempty"`
	ReadIdleTimeout     uint64 `json:"read_idle_timeout,omitempty"`
	Heartbeat           uint64 `json:"health_check_timeout,omitempty"`
	DisableMTUDiscovery bool   `json:"disable_mtu_discovery,omitempty"`
	Obfs                string `json:"obfs,omitempty"`
	Password            string `json:"password,omitempty"`
}
