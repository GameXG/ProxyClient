package proxyclient

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"reflect"
	"testing"
	"time"

	"fmt"

	"strconv"

	"gopkg.in/bufio.v1"
)

func testSocks5ProixyServer(t *testing.T, proxyAddr string, usernameAndPassword []byte, attypAddr []byte, port uint16, ci chan int) {
	b := make([]byte, 30)
	l, err := net.Listen("tcp", proxyAddr)
	if err != nil {
		t.Fatalf("错误,%v", err)
	}

	ci <- 1

	c, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}

	if len(usernameAndPassword) == 0 {
		if n, err := c.Read(b); err != nil || bytes.Equal(b[:n], []byte{0x05, 0x01, 0x00}) != true {
			t.Fatal("鉴定请求错误：", err)
		}

		if _, err := c.Write([]byte{0x05, 0x00}); err != nil {
			t.Fatalf("回应鉴定错误：%v", err)
		}
	} else {
		if n, err := c.Read(b); err != nil || bytes.Equal(b[:n], []byte{0x05, 0x01, 0x02}) != true {
			t.Fatalf("鉴定请求错误：%v", err)
		}

		if _, err := c.Write([]byte{0x05, 0x02}); err != nil {
			t.Fatalf("回应鉴定错误：%v", err)
		}

		if n, err := c.Read(b); err != nil || b[0] != 0x01 || bytes.Equal(b[1:n], usernameAndPassword) != true {
			t.Fatalf("用户名密码错误：%v", err)
		}

		if _, err := c.Write([]byte{0x01, 0x00}); err != nil {
			t.Fatalf("回应登陆错误：%v", err)
		}
	}

	// 构建应该受到的请求内容
	br := make([]byte, 5+len(attypAddr))
	n := copy(br, []byte{0x05, 0x01, 0x00})
	n = copy(br[n:], attypAddr)
	binary.BigEndian.PutUint16(br[n+3:], port)

	// 接收命令请求
	if n, err := c.Read(b); err != nil || bytes.Equal(b[:n], br) != true {
		t.Fatalf("请求命令错误：%v,%v!=%v", err, br, b[:n])
	}

	// 发出回应
	if _, err := c.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x1, 0x2, 0x3, 0x4, 0x80, 0x80}); err != nil {
		t.Fatalf("请求回应错误：%v", err)
	}

	if n, err := c.Read(b); err != nil || bytes.Equal(b[:n], B1) != true {
		t.Fatalf("B1不正确。err=%v，B1=%v,b=%v", err, B1, b[:n])
	}

	// 发出B2
	if _, err := c.Write(B2); err != nil {
		t.Fatalf("B2 发送错误：%v", err)
	}

	if v, ok := c.(TCPConn); ok != true {
		t.Fatalf("类型不匹配错误。")
	} else {
		v.SetLinger(5)
	}
	c.Close()

}

func testSocks5ProxyClient(t *testing.T, proxyAddr string, addr string) {
	b := make([]byte, 30)
	p, err := NewProxyClient(proxyAddr)
	if err != nil {
		t.Fatal("启动代理错误:", err)
	}

	c, err := p.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatal("通过代理建立连接错误：", err, "proxyAddr:", proxyAddr, "addr:", addr)
	}

	// 发出B1
	if _, err := c.Write(B1); err != nil {
		t.Fatal("B1 发送错误：", err)
	}

	//接收B2
	if n, err := c.Read(b); err != nil || bytes.Equal(b[:n], B2) != true {
		t.Fatalf("B2不正确。err=%v，B1=%v,b=%v", err, B2, b[:n])
	}

	if _, err := c.Read(b); err != io.EOF {
		t.Fatal("读EOF错误。err=", err)
	}
}

func TestSocks5Proxy(t *testing.T) {
	ci := make(chan int)
	b := make([]byte, 0, 30)

	// 测试域名
	addr := "www.163.com"

	b = append(b, 0x03, byte(len(addr)))
	b = append(b, []byte(addr)...)

	go testSocks5ProixyServer(t, "127.0.0.1:13337", nil, b, 80, ci)
	<-ci
	testSocks5ProxyClient(t, "socks5://127.0.0.1:13337", "www.163.com:80")

	// 测试 ipv4
	addr = "1.2.3.4"
	b = b[0:0]
	b = append(b, 0x01)
	b = append(b, []byte(net.ParseIP(addr).To4())...)

	go testSocks5ProixyServer(t, "127.0.0.1:13338", nil, b, 80, ci)
	<-ci
	testSocks5ProxyClient(t, "socks5://127.0.0.1:13338", "1.2.3.4:80")

	// 测试 ipv6
	addr = "1:2:3:4::5:6"
	b = b[0:0]
	b = append(b, 0x04)
	b = append(b, []byte(net.ParseIP(addr))...)

	go testSocks5ProixyServer(t, "127.0.0.1:13339", nil, b, 80, ci)
	<-ci
	testSocks5ProxyClient(t, "socks5://127.0.0.1:13339", "[1:2:3:4::5:6]:80")
}

func TestSocks4Proxy(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	sAddr := l.Addr().String()

	test := func(addr string, proxy string) {
		p, err := NewProxyClient(fmt.Sprint(proxy))
		if err != nil {
			t.Error(err)
			return
		}

		go func() {
			dstIp, dstHost, dstPort, err := Socks4Server(t, l)
			if err != nil {
				t.Error(err)
			}

			host, port, err := net.SplitHostPort(sAddr)
			ip := net.ParseIP(host)

			if len(ip) == 0 {
				if host != dstHost {
					t.Error("host!=dstHost")
					return
				}
			} else {
				ip = ip.To4()

				if !reflect.DeepEqual(ip, dstIp) {
					t.Error("ip!=dstIp")
					return
				}
			}

			if port != strconv.Itoa(dstPort) {
				t.Error("port!=dstPort")
				return
			}
		}()

		c, err := p.Dial("tcp", addr)
		if err != nil {
			t.Error(err)
			return
		}

		data := []byte{0, 1, 2, 3, 4}
		_, err = c.Write(data)
		if err != nil {
			t.Error(err)
			return
		}

		b := make([]byte, len(data))
		_, err = io.ReadFull(c, b)
		if err != nil {
			t.Error(err)
			return
		}

		time.Sleep(1 * time.Second)

		if !reflect.DeepEqual(data, b) {
			t.Errorf("%#v!=%#v", data, b)
			return
		}

	}

	// 测试 ip 地址
	test("1.2.3.4:80", fmt.Sprint("socks4://", sAddr))
	test("1.2.3.4:80", fmt.Sprint("socks4a://", sAddr))

	// 测试 域名
	test("www.aaa.com:80", fmt.Sprint("socks4a://", sAddr))
}

func Socks4Server(t *testing.T, l net.Listener) (dstIp net.IP, dstHost string, dstPort int, err error) {
	/*
		l, err := net.Listen("tcp", "127.0.0.0:0")
		if err != nil {
			return "", err
		}
		addr = l.Addr().String()
	*/
	cmd := make([]byte, 9, 100)

	var c net.Conn
	c, err = l.Accept()
	if err != nil {
		t.Error(err)
		return
	}
	defer c.Close()

	r := bufio.NewReader(c)

	_, err = io.ReadFull(r, cmd)
	if err != nil {
		t.Error(err)
		return
	}

	vn := cmd[0]
	cd := cmd[1]
	dstport := cmd[2 : 2+2]
	dstip := cmd[4 : 4+4]
	null := cmd[8]

	if vn != 4 || cd != 1 {
		t.Error("vn!=4 || cd!=1")
		return
	}

	dstPort = int(binary.BigEndian.Uint16(dstport))

	if null != 0 {
		t.Error("null!=0")
		return
	}

	if reflect.DeepEqual(dstip[:3], []byte{0, 0, 0}) {
		// 域名
		var b []byte
		b, err = r.ReadSlice(0)
		if err != nil {
			t.Error(err)
			return
		}

		if b[len(b)-1] != 0 {
			t.Fatal("b[len(b)-1:]!=0")
			return
		}

		dstHost = string(b[:len(b)-1])
	} else {
		dstIp = net.IP(dstip)
	}

	buf := []byte{0, 90}
	buf = append(buf, dstport...)
	buf = append(buf, dstip...)
	_, err = c.Write(buf)
	if err != nil {
		t.Error(err)
		return
	}

	_, err = io.Copy(c, r)
	if err == io.EOF {
		err = nil
	}

	return
}
