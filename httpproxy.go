package proxyclient

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type HttpTCPConn struct {
	Conn                            //http 协议时是原始链接、https协议时是tls.Conn
	rawConn               TCPConn   //原始链接
	tlsConn               *tls.Conn //tls链接
	localAddr, remoteAddr net.TCPAddr
	localHost, remoteHost string
	LocalPort, remotePort uint16
	proxyClient           ProxyClient
	r                     io.Reader
}

type httpProxyClient struct {
	proxyAddr   string
	proxyDomain string // 用于ssl证书验证
	proxyType   string // socks4 socks5
	//TODO: 用户名、密码
	insecureSkipVerify bool
	upProxy            ProxyClient
}

// 创建代理客户端
// ProxyType				http https
// ProxyAddr 				127.0.0.1:5555
// proxyDomain				ssl 验证域名，"" 则使用 proxyAddr 部分的域名
// insecureSkipVerify		使用https代理时是否忽略证书检查
// UpProxy
func NewHttpProxyClient(proxyType string, proxyAddr string, proxyDomain string, insecureSkipVerify bool, upProxy ProxyClient) (ProxyClient, error) {
	proxyType = strings.ToLower(strings.Trim(proxyType, " \r\n\t"))
	if proxyType != "http" && proxyType != "https" {
		return nil, errors.New("ProxyType 错误的格式，只支持http、https代理。")
	}

	if upProxy == nil {
		nUpProxy, err := NewDriectProxyClient("")
		if err != nil {
			return nil, fmt.Errorf("创建直连代理错误：%v", err)
		}
		upProxy = nUpProxy
	}

	if proxyDomain == "" {
		host, _, err := net.SplitHostPort(proxyAddr)
		if err != nil {
			return nil, fmt.Errorf("proxyAddr 格式错误：%v", err)
		}
		proxyDomain = host
	}

	return &httpProxyClient{proxyAddr, proxyDomain, proxyType, insecureSkipVerify, upProxy}, nil
}

func (p *httpProxyClient) Dial(network, address string) (Conn, error) {
	if strings.HasPrefix(strings.ToLower(network), "tcp") {
		return p.DialTCPSAddr(network, address)
	} else if strings.HasPrefix(strings.ToLower(network), "udp") {
		return nil, errors.New("不支持UDP协议。")
	} else {
		return nil, errors.New("未知的 network 类型。")
	}
}

func (p *httpProxyClient) DialTimeout(network, address string, timeout time.Duration) (Conn, error) {
	return nil, errors.New("暂不支持")
}

func (p *httpProxyClient) DialTCP(network string, laddr, raddr *net.TCPAddr) (ProxyTCPConn, error) {
	if laddr != nil || laddr.Port != 0 {
		return nil, errors.New("代理协议不支持指定本地地址。")
	}

	return p.DialTCPSAddr(network, raddr.String())
}

func (p *httpProxyClient) DialTCPSAddr(network string, raddr string) (ProxyTCPConn, error) {
	var tlsConn *tls.Conn
	rawConn, err := p.upProxy.DialTCPSAddr(network, p.proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("无法连接代理服务器 %v ，错误：%v", p.proxyAddr, err)
	}

	var c Conn = rawConn

	if p.proxyType == "https" {
		tlsConn = tls.Client(c, &tls.Config{ServerName: p.proxyDomain, InsecureSkipVerify: p.insecureSkipVerify})
		if err := tlsConn.Handshake(); err != nil {
			tlsConn.Close()
			return nil, fmt.Errorf("TLS 协议握手错误：%v", err)
		}
		if p.insecureSkipVerify == false && tlsConn.VerifyHostname(p.proxyDomain) != nil {
			tlsConn.Close()
			return nil, fmt.Errorf("TLS 协议域名验证失败：%v", err)
		}
		c = tlsConn
	}

	req, err := http.NewRequest("CONNECT", p.proxyAddr, nil)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("创建请求错误：%v", err)
	}
	req.URL.Path = raddr
	req.URL.Host = p.proxyAddr

	if err := req.Write(c); err != nil {
		c.Close()
		return nil, fmt.Errorf("写请求错误：%v", err)
	}

	br := bufio.NewReader(c)

	res, err := http.ReadResponse(br, req)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("响应格式错误：%v", err)
	}

	if res.StatusCode != 200 {
		c.Close()
		return nil, fmt.Errorf("响应错误：%v", res)
	}

	return &HttpTCPConn{c, rawConn, tlsConn, net.TCPAddr{}, net.TCPAddr{}, "", "", 0, 0, p, res.Body}, nil
}

// 重写了 Read 接口
// 由于 http 协议问题，解析响应需要读缓冲，所以必须重写 Read 来兼容读缓冲功能。
func (c *HttpTCPConn) Read(b []byte) (n int, err error) {
	return c.r.Read(b)
}

func (c *HttpTCPConn) SetLinger(sec int) error {
	return c.rawConn.SetLinger(sec)
}

func (c *HttpTCPConn) SetNoDelay(noDelay bool) error {
	return c.rawConn.SetNoDelay(noDelay)
}
func (c *HttpTCPConn) SetReadBuffer(bytes int) error {
	return c.rawConn.SetReadBuffer(bytes)
}
func (c *HttpTCPConn) SetWriteBuffer(bytes int) error {
	return c.rawConn.SetWriteBuffer(bytes)
}

func (p *httpProxyClient) DialUDP(network string, laddr, raddr *net.UDPAddr) (ProxyUDPConn, error) {
	return nil, fmt.Errorf("%v 代理不支持 UDP 转发。", p.proxyType)
}

func (p *httpProxyClient) UpProxy() ProxyClient {
	return p.upProxy
}

func (p *httpProxyClient) SetUpProxy(upProxy ProxyClient) error {
	p.upProxy = upProxy
	return nil
}

func (c *HttpTCPConn) ProxyClient() ProxyClient {
	return c.proxyClient
}
