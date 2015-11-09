package proxyclient

import (
	"testing"
	"bytes"
	"io"
	"net"
	"fmt"
)

const (CONNECT = "CONNECT")

// 伪装成为一个代理服务器。
func testHttpProixyServer(t *testing.T, proxyAddr string, rAddr string, ci chan int) {

	l, err := net.Listen("tcp", proxyAddr)
	if err != nil {
		fmt.Println("监听错误:%v", err)
		t.Fatal("监听错误:%v", err)
	}

	ci <- 1
	c, err := l.Accept()
	if err != nil {
		t.Fatal("接受连接错误:", err)
	}

	b := make([]byte, 1024)

	if n, err := c.Read(b); err != nil {
		t.Fatal("读错误：%v", err)
	}else {
		b = b[:n]
		print(b)
	}

	connect := CONNECT + " " + rAddr

	if bytes.Equal(b[:len(connect)], []byte(connect)) != true {
		t.Fatal("命令不匹配！")
	}

	if _, err := c.Write([]byte("HTTP/1.0 200 ok\r\nAAA:111\r\n\r\n")); err != nil {
		t.Fatal("写数据错误")
	}


	if n, err := c.Read(b[:1024]); err != nil {
		t.Fatal("读错误：%v", err)
	}else {
		b = b[:n]
		print(b)
	}


	if _, err := c.Write([]byte("HTTP/1.0 200 ok\r\nHead1:11111\r\n\r\nHello Word!")); err != nil {
		t.Fatal("写数据错误")
	}

	c.Close()



}

func TestHttpProxy(t *testing.T) {
	ci := make(chan int)
	go testHttpProixyServer(t, "127.0.0.1:1331", "www.google.com:80", ci)
	<-ci

	p, err := NewProxyClient("http://127.0.0.1:1331")
	if err != nil {
		t.Fatal("连接代理服务器错误：%v", err)
	}

	c, err := p.Dial("tcp", "www.google.com:80")
	if err != nil {
		t.Fatal("通过代理服务器连接目标网站失败：%v", err)
	}

	if _, err := c.Write([]byte("GET / HTTP/1.0\r\nHOST:www.google.com\r\n\r\n")); err != nil {
		t.Fatal("请求发送错误：%v", err)
	}

	b := make([]byte, 1024)
	if n, err := c.Read(b); err != nil {
		t.Fatal("响应读取错误：%v", err)
	}else {
		b = b[:n]
	}

	if bytes.Equal(b, []byte("HTTP/1.0 200 ok\r\nHead1:11111\r\n\r\nHello Word!")) != true {
		t.Fatal("返回内容不匹配：%v", string(b))
	}

	if _, err:= c.Read(b[:1024]); err != io.EOF {
		t.Fatal("非预期的结尾：%v", err)
	}

}

