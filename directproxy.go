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
}

// 创建代理客户端
// 直连 direct://0.0.0.0:0000/?LocalAddr=123.123.123.123:0
func NewDriectProxyClient(localAddr string) (ProxyClient, error) {
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

	return &directProxyClient{*tcpAddr, *udpAddr}, nil
}

func (p *directProxyClient) Dial(network, address string) (Conn, error) {
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

func (p *directProxyClient) DialTimeout(network, address string, timeout time.Duration) (Conn, error) {
	return nil, errors.New("未完成")
}
func (p *directProxyClient) DialTCP(network string, laddr, raddr *net.TCPAddr) (ProxyTCPConn, error) {
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
	addr, err := net.ResolveTCPAddr(network, raddr)
	if err != nil {
		return nil, fmt.Errorf("代理服务器地址错误，无法解析：%v", err)
	}
	return p.DialTCP(network, nil, addr)
}

func (p *directProxyClient) DialUDP(network string, laddr, raddr *net.UDPAddr) (ProxyUDPConn, error) {
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
