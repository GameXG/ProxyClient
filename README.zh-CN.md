# ProxyClient

[![Build Status](https://travis-ci.org/GameXG/ProxyClient.svg?branch=master)](https://travis-ci.org/GameXG/ProxyClient)

**[English](README.md) | 简体中文**

ProxyClient 是一个 Go 代理客户端，提供接近标准库 `net` 包的 API。

支持代理级联。原生支持 SOCKS4、SOCKS4A、SOCKS5、HTTP、HTTPS 和直连；`shadowsocks`、`vmess`、`vless`、`trojan`、`wireguard`、`hysteria2` 通过内置的 `xray-core` 协议引擎处理，同时兼容 `ss`、`hysteria` 和 `hy2` 别名。

TLS 默认启用证书校验。新版 Xray 不再支持 `allowInsecure`。自签名证书请在 URL 查询参数中使用 `pcs` / `pinnedPeerCertSha256` 或 `vcn` / `verifyPeerCertByName`。显式启用旧版 `allowInsecure` 参数时，必须同时提供其中一种安全校验方式。

```go
package main

import (
	"fmt"
	"io"

	proxyclient "github.com/GameXG/ProxyClient"
)

func main() {
	p, err := proxyclient.NewProxyClient("socks5://127.0.0.1:5556?upProxy=https://145.2.1.3:8080")
	if err != nil {
		panic("创建代理客户端失败")
	}

	c, err := p.Dial("tcp", "www.google.com:80")
	if err != nil {
		panic("连接失败")
	}
	defer c.Close()

	io.WriteString(c, "GET / HTTP/1.0\r\nHost: www.google.com\r\n\r\n")
	b, err := io.ReadAll(c)
	if err != nil {
		panic("读取失败")
	}
	fmt.Print(string(b))
}
```

## API

客户端提供兼容 `net.Conn` 的连接接口，并通过 `SetUpProxy` 支持代理级联。

```go
type Conn interface {
	net.Conn
}

type TCPConn interface {
	Conn
	SetLinger(sec int) error
	SetNoDelay(noDelay bool) error
	SetReadBuffer(bytes int) error
	SetWriteBuffer(bytes int) error
}

type ProxyTCPConn interface {
	TCPConn
	ProxyClient() ProxyClient
}

type UDPConn interface {
	Conn
}

type ProxyUDPConn interface {
	UDPConn
	ProxyClient() ProxyClient
}

type ProxyClient interface {
	UpProxy() ProxyClient
	SetUpProxy(upProxy ProxyClient) error

	Dial(network, address string) (net.Conn, error)
	DialTimeout(network, address string, timeout time.Duration) (net.Conn, error)
	DialTCP(network string, laddr, raddr *net.TCPAddr) (net.Conn, error)
	DialTCPSAddr(network string, raddr string) (ProxyTCPConn, error)
	DialTCPSAddrTimeout(network string, raddr string, timeout time.Duration) (ProxyTCPConn, error)
	DialUDP(network string, laddr, raddr *net.UDPAddr) (net.Conn, error)
	GetProxyAddrQuery() map[string][]string
}
```

使用 `NewProxyClient` 创建客户端，支持以下 URL 格式：

```text
http://123.123.123.123:8088
https://123.123.123.123:8088
socks4://123.123.123.123:5050
socks4a://123.123.123.123:5050
socks5://123.123.123.123:5050?upProxy=http://145.2.1.3:8080
shadowsocks://...
vmess://...
vless://...
trojan://...
wireguard://...
hysteria2://...
direct://0.0.0.0:0000/?localAddr=123.123.123.123:0
```

兼容别名：`ss://`、`hysteria://` 和 `hy2://`。

## 致谢

扩展协议能力由内置的 [`xray-core`](https://github.com/XTLS/Xray-core) 引擎提供。
