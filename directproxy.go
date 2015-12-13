package proxyclient

import (
	"net"

	"errors"
	"fmt"
	"strings"
	"time"
)

type DirectTCPConn struct {
	net.TCPConn
	proxyClient ProxyClient
}

type DirectUDPConn struct {
	net.UDPConn
	proxyClient ProxyClient
}
type directProxyClient struct {
	TCPLocalAddr net.TCPAddr
	UDPLocalAddr net.UDPAddr
	query        map[string][]string
}

// 创建代理客户端
// 直连 direct://0.0.0.0:0000/?LocalAddr=123.123.123.123:0
func NewDriectProxyClient(localAddr string, query map[string][]string) (ProxyClient, error) {
	if localAddr == "" {
		localAddr = "0.0.0.0:0"
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", localAddr)
	if err != nil {
		return nil, errors.New("LocalAddr 错误的格式")
	}

	udpAddr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return nil, errors.New("LocalAddr 错误的格式")
	}

	return &directProxyClient{*tcpAddr, *udpAddr, query}, nil
}

func (p *directProxyClient) Dial(network, address string) (net.Conn, error) {
	if strings.HasPrefix(network, "tcp") {
		addr, err := net.ResolveTCPAddr(network, address)
		if err != nil {
			return nil, fmt.Errorf("地址解析错误:%v", err)
		}
		return p.DialTCP(network, &p.TCPLocalAddr, addr)
	} else if strings.HasPrefix(network, "udp") {
		addr, err := net.ResolveUDPAddr(network, address)
		if err != nil {
			return nil, fmt.Errorf("地址解析错误:%v", err)
		}
		return p.DialUDP(network, &p.UDPLocalAddr, addr)
	} else {
		return nil, errors.New("未知的 network 类型。")
	}
}

func (p *directProxyClient) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
	case "udp", "udp4", "udp6":
	default:
		return nil, fmt.Errorf("不支持的 network 类型:%v", network)
	}

	d := net.Dialer{Timeout:timeout, LocalAddr:&p.TCPLocalAddr}
	conn, err := d.Dial(network, address)
	if err != nil {
		return nil, err
	}

	switch conn := conn.(type) {
	case *net.TCPConn:
		return &DirectTCPConn{*conn, p}, nil
	case *net.UDPConn:
		return &DirectUDPConn{*conn, p}, nil
	default:
		return nil, fmt.Errorf("内部错误：未知的连接类型。")
	}
}

func (p *directProxyClient) DialTCP(network string, laddr, raddr *net.TCPAddr) (net.Conn, error) {
	if laddr == nil {
		laddr = &p.TCPLocalAddr
	}
	conn, err := net.DialTCP(network, laddr, raddr)
	if err != nil {
		return nil, err
	}
	return &DirectTCPConn{*conn, p}, nil
}

func (p *directProxyClient)DialTCPSAddr(network string, raddr string) (ProxyTCPConn, error) {
	return p.DialTCPSAddrTimeout(network, raddr, 0)
}

// DialTCPSAddrTimeout 同 DialTCPSAddr 函数，增加了超时功能
func (p *directProxyClient)DialTCPSAddrTimeout(network string, raddr string, timeout time.Duration) (rconn ProxyTCPConn, rerr error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, fmt.Errorf("不支持的 network 类型:%v", network)
	}
	d := net.Dialer{Timeout:timeout, LocalAddr:&p.TCPLocalAddr}
	conn, err := d.Dial(network, raddr)
	if err != nil {
		return nil, err
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		return &DirectTCPConn{*tcpConn, p}, nil
	}
	return nil, fmt.Errorf("内部错误")
}

func (p *directProxyClient) DialUDP(network string, laddr, raddr *net.UDPAddr) (net.Conn, error) {
	if laddr == nil {
		laddr = &p.UDPLocalAddr
	}
	conn, err := net.DialUDP(network, laddr, raddr)
	if err != nil {
		return nil, err
	}
	return &DirectUDPConn{*conn, p}, nil
}
func (p *directProxyClient) UpProxy() ProxyClient {
	return nil
}
func (p *directProxyClient) SetUpProxy(upProxy ProxyClient) error {
	return errors.New("直连不支持上层代理。")
}
func (c *DirectTCPConn) ProxyClient() ProxyClient {
	return c.proxyClient
}
func (c *DirectUDPConn) ProxyClient() ProxyClient {
	return c.proxyClient
}
func (c *directProxyClient)GetProxyAddrQuery() map[string][]string {
	return c.query
}