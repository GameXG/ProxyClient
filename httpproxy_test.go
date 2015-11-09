package proxyclient

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
)

func testHttpProixyServer(t *testing.T, proxyAddr string, rAddr string, ci chan int) {
	b := make([]byte, 1024)
	addr, err := net.ResolveTCPAddr("tcp", proxyAddr)
	if err != nil {
		t.Errorf("错误的地址(%v)：%v", proxyAddr, err)
		return
	}

	listen, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Errorf("错误,%v", err)
		return
	}

	ci <- 1

	c, err := listen.Accept()

	if v, ok := c.(TCPConn); ok != true {
		t.Errorf("S tcp 连接错误类型")
		return
	} else {
		v.SetLinger(5)
		//	v.SetDeadline(time.Now().Add(3 * time.Second))
	}

	if n, err := c.Read(b); err != nil {
		fmt.Print(err)
		t.Errorf("读错误：err=%v ,b=%v", err, b[:n])
	} else {
		b = b[:n]
		str := string(b)
		if strings.HasPrefix(str, "CONNECT "+rAddr) != true {
			t.Errorf("请求未包含:%v", rAddr)
		}
	}

	if _, err := c.Write([]byte("HTTP/1.0 200 conn ok\r\nProxy-agent:XXX\r\n\r\n")); err != nil {
		t.Errorf("写错误：%v", err)
		return
	} else {
		fmt.Println("S已发出 200 回应")
	}

	if n, err := c.Read(b[:1024]); err != nil {
		fmt.Println("S读数据错误", n, err)
		t.Errorf("S读数据错误，err=%v, data=%v", err, b[:n])
		return
	} else {
		fmt.Println("S读到数据：", b)
		if bytes.Equal(b[:n], []byte("0123456789")) != true {
			fmt.Println("S读数据不匹配：", b)
		} else {
			fmt.Print("S读数据匹配")
		}
	}

	if _, err := c.Write([]byte("9876543210")); err != nil {
		t.Errorf("S写错误：%v", err)
		return
	} else {
		fmt.Println("S写数据：9876543210")
	}

	c.Close()

}

func TestHttpProxy(t *testing.T) {

	ci := make(chan int)
	go testHttpProixyServer(t, "127.0.0.1:13340", "www.google.com:80", ci)
	<-ci

	p, err := NewHttpProxyClient("http", "127.0.0.1:13340", "", false, nil)
	if err != nil {
		t.Errorf("HTTP 代理建立失败：%v", err)
		return
	}

	c, err := p.Dial("tcp", "www.google.com:80")
	if err != nil {
		t.Errorf("C HTTP代理连接远程服务器失败。")
		return
	} else {
		fmt.Println("C连接已建立")
	}

	if _, err := c.Write([]byte("0123456789")); err != nil {
		t.Errorf("C写错误：%v", err)
		return
	} else {
		fmt.Println("C 0123456789 已发出")
	}

	b := make([]byte, 50)
	// 卡到了这里。
	if n, err := c.Read(b); err != nil {
		t.Errorf("C 读错误：err:%v,b:%v", err, b[:n])
		return
	} else {
		b = b[:n]
		fmt.Println("C 已读到：", b)
	}
	if bytes.Equal(b, []byte("9876543210")) != true {
		t.Errorf("C 读数据不匹配")
		return
	} else {
		fmt.Println("C 读数据正确")
	}

	if _, err := c.Read(b); err != io.EOF {
		t.Errorf("C 非预期额结尾：%v", err)
		return
	} else {
		fmt.Println("C 到达预期结尾")
	}
}
