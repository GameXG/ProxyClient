package proxyclient

import (
	"fmt"
	xrayclient "github.com/GameXG/ProxyClient/xray"
	"net"
	"net/url"
	"strings"
	"time"
)

type xrayProxyClient struct {
	rawURL string
	query  map[string][]string
	inner  ProxyClient
}

func canonicalProtocolScheme(scheme string) string {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "ss", "shadowsocks":
		return "shadowsocks"
	case "hysteria", "hysteria2", "hy2":
		return "hysteria2"
	default:
		return strings.ToLower(strings.TrimSpace(scheme))
	}
}

func NewXrayProxyClient(u *url.URL, query map[string][]string) (ProxyClient, error) {
	if u == nil {
		return nil, fmt.Errorf("xray 节点地址不能为空")
	}

	var (
		port int
		err  error
	)

	switch canonicalProtocolScheme(u.Scheme) {
	case "shadowsocks":
		_, port, err = xrayclient.StartShadowsocks(u, 0)
	case "vmess":
		_, port, err = xrayclient.StartVmess(u, 0)
	case "vless":
		_, port, err = xrayclient.StartVless(u, 0)
	case "trojan":
		_, port, err = xrayclient.StartTrojan(u, 0)
	case "wireguard":
		_, port, err = xrayclient.StartWireGuard(u, 0)
	case "hysteria2":
		_, port, err = xrayclient.StartHysteria(u, 0)
	default:
		return nil, fmt.Errorf("不支持的扩展协议: %s", u.Scheme)
	}
	if err != nil {
		return nil, fmt.Errorf("启动扩展协议失败(%s): %w", u.Scheme, err)
	}
	if port <= 0 {
		return nil, fmt.Errorf("分配本地 socks5 端口失败")
	}

	inner, err := NewSocksProxyClient("socks5", fmt.Sprintf("127.0.0.1:%d", port), nil, query)
	if err != nil {
		return nil, fmt.Errorf("创建本地 socks5 代理失败: %w", err)
	}

	return &xrayProxyClient{
		rawURL: u.String(),
		query:  query,
		inner:  inner,
	}, nil
}

func (p *xrayProxyClient) UpProxy() ProxyClient {
	return p.inner.UpProxy()
}

func (p *xrayProxyClient) SetUpProxy(upProxy ProxyClient) error {
	return p.inner.SetUpProxy(upProxy)
}

func (p *xrayProxyClient) Dial(network, address string) (net.Conn, error) {
	return p.inner.Dial(network, address)
}

func (p *xrayProxyClient) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return p.inner.DialTimeout(network, address, timeout)
}

func (p *xrayProxyClient) DialTCP(network string, laddr, raddr *net.TCPAddr) (net.Conn, error) {
	return p.inner.DialTCP(network, laddr, raddr)
}

func (p *xrayProxyClient) DialTCPSAddr(network string, raddr string) (ProxyTCPConn, error) {
	return p.inner.DialTCPSAddr(network, raddr)
}

func (p *xrayProxyClient) DialTCPSAddrTimeout(network string, raddr string, timeout time.Duration) (ProxyTCPConn, error) {
	return p.inner.DialTCPSAddrTimeout(network, raddr, timeout)
}

func (p *xrayProxyClient) DialUDP(network string, laddr, raddr *net.UDPAddr) (net.Conn, error) {
	return p.inner.DialUDP(network, laddr, raddr)
}

func (p *xrayProxyClient) GetProxyAddrQuery() map[string][]string {
	return p.query
}
