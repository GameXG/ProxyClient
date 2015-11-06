package proxyclient

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
)

func testSocks5ProixyServer(t *testing.T, proxyAddr string, attypAddr []byte, port uint16, ci chan int) {
	b := make([]byte, 30)
	l, err := net.Listen("tcp", proxyAddr)
	if err != nil {
		t.Errorf("错误,%v", err)
	}

	ci <- 1

	c, err := l.Accept()

	if n, err := c.Read(b); err != nil || bytes.Equal(b[:n], []byte{0x05, 0x01, 0x00}) != true {
		t.Errorf("鉴定请求错误：%v", err)
	}

	if _, err := c.Write([]byte{0x05, 0x00}); err != nil {
		t.Errorf("回应鉴定错误：%v", err)
	}

	// 构建应该受到的请求内容
	br := make([]byte, 5+len(attypAddr))
	n := copy(br, []byte{0x05, 0x01, 0x00})
	n = copy(br[n:], attypAddr)
	binary.BigEndian.PutUint16(br[n+3:], port)

	// 接收命令请求
	if n, err := c.Read(b); err != nil || bytes.Equal(b[:n], br) != true {
		t.Errorf("请求命令错误：%v,%v!=%v", err, br, b[:n])
	}

	// 发出回应
	if _, err := c.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x1, 0x2, 0x3, 0x4, 0x80, 0x80}); err != nil {
		t.Errorf("请求回应错误：%v", err)
	}

	if n, err := c.Read(b); err != nil || bytes.Equal(b[:n], B1) != true {
		t.Errorf("B1不正确。err=%v，B1=%v,b=%v", err, B1, b[:n])
	}

	// 发出B2
	if _, err := c.Write(B2); err != nil {
		t.Errorf("B2 发送错误：%v", err)
	}

	if v, ok := c.(*net.TCPConn); ok != true {
		t.Errorf("类型不匹配错误。")
	} else {
		v.SetLinger(5)
	}
	c.Close()

}

func testSocks5ProxyClient(t *testing.T, proxyAddr string, addr string) {
	b := make([]byte, 30)
	p, err := NewSocksProxyClient("socks5", proxyAddr, nil)
	if err != nil {
		t.Errorf("启动代理错误:%v", err)
	}
	c, err := p.Dial("tcp", addr)
	if err != nil {
		t.Errorf("通过代理建立连接错误：%v", err)
	}

	// 发出B1
	if _, err := c.Write(B1); err != nil {
		t.Errorf("B1 发送错误：%v", err)
	}

	//接收B2
	if n, err := c.Read(b); err != nil || bytes.Equal(b[:n], B2) != true {
		t.Errorf("B2不正确。err=%v，B1=%v,b=%v", err, B2, b[:n])
	}

	if _, err := c.Read(b); err != io.EOF {
		t.Errorf("读EOF错误。err=%", err)
	}
}

func TestSocksProxy(t *testing.T) {
	ci := make(chan int)
	b := make([]byte, 0, 30)

	// 测试域名
	addr := "www.163.com"

	b = append(b, 0x03, byte(len(addr)))
	b = append(b, []byte(addr)...)

	go testSocks5ProixyServer(t, "127.0.0.1:13337", b, 80, ci)
	<-ci
	testSocks5ProxyClient(t, "127.0.0.1:13337", "www.163.com:80")

	// 测试 ipv4
	addr = "1.2.3.4"
	b = b[0:0]
	b = append(b, 0x01)
	b = append(b, []byte(net.ParseIP(addr).To4())...)

	go testSocks5ProixyServer(t, "127.0.0.1:13338", b, 80, ci)
	<-ci
	testSocks5ProxyClient(t, "127.0.0.1:13338", "1.2.3.4:80")

	// 测试 ipv6
	addr = "1:2:3:4::5:6"
	b = b[0:0]
	b = append(b, 0x04)
	b = append(b, []byte(net.ParseIP(addr))...)

	go testSocks5ProixyServer(t, "127.0.0.1:13339", b, 80, ci)
	<-ci
	testSocks5ProxyClient(t, "127.0.0.1:13339", "[1:2:3:4::5:6]:80")

}
