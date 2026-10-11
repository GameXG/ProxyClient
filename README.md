# ProxyClient
[![Build Status](https://travis-ci.org/GameXG/ProxyClient.svg?branch=master)](https://travis-ci.org/GameXG/ProxyClient)

**English | [简体中文](README.zh-CN.md)**

ProxyClient is a Go proxy client with an API modeled after the standard `net` package.

It supports proxy chaining. SOCKS4, SOCKS4A, SOCKS5, HTTP, HTTPS, and direct connections are implemented natively. `shadowsocks`, `vmess`, `vless`, `trojan`, `wireguard`, and `hysteria2` are handled by the bundled `xray-core` protocol engine, with `ss`, `hysteria`, and `hy2` aliases supported.

TLS certificate verification is enabled by default. Newer Xray versions no longer support `allowInsecure`. For self-signed certificates, use the `pcs` / `pinnedPeerCertSha256` or `vcn` / `verifyPeerCertByName` URL query parameters. An explicitly enabled legacy `allowInsecure` parameter must be accompanied by one of these secure verification options.

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
		panic("failed to create proxy client")
	}

	c, err := p.Dial("tcp", "www.google.com:80")
	if err != nil {
		panic("connection failed")
	}
	defer c.Close()

	io.WriteString(c, "GET / HTTP/1.0\r\nHost: www.google.com\r\n\r\n")
	b, err := io.ReadAll(c)
	if err != nil {
		panic("read failed")
	}
	fmt.Print(string(b))
}
```

## API

The client exposes connection interfaces compatible with `net.Conn` and supports proxy chaining through `SetUpProxy`.

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

Create a client with `NewProxyClient` using one of these URL forms:

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

Aliases: `ss://`, `hysteria://`, and `hy2://`.

## Credits

Extension protocol support is provided by the bundled [`xray-core`](https://github.com/XTLS/Xray-core) engine.
