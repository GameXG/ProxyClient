package proxyclient

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"testing"
)

var B1 = []byte{15, 63, 25, 41, 63, 48, 45, 32, 14}
var B2 = []byte{41, 68, 23, 94, 14, 86, 42, 56}
var B3 = []byte{47, 65, 36, 14, 89, 96, 32, 14, 56}

func testDirectProxyTCP1(t *testing.T) {

	p, err := NewDriectProxyClient("",make(map[string][]string))
	if err != nil {
		t.Errorf("启动直连代理失败：%s", err)
		fmt.Printf("%v %v", p, err)
		return
	}

	b := make([]byte, 30)
	c, err := p.Dial("tcp", "127.0.0.1:13336")
	if err != nil {
		t.Errorf("代理连接失败：%v", err)
	} else {
		fmt.Println("连接成功")
	}
	if _, err := c.Read(b); err != nil {
		t.Errorf("代理读错误：%v", err)
	} else {
		fmt.Println("读b1成功")
	}
	if bytes.Equal(b[:len(B1)], B1) != true {
		t.Errorf("数据不匹配：%v", err)
	}

	if _, err := c.Write(B2); err != nil {
		t.Errorf("写数据失败：%v", err)
	} else {
		fmt.Println("写b2成功")
	}

	if _, err := c.Read(b); err != nil {
		t.Errorf("代理读错误：%v", err)
	} else {
		fmt.Println("读b3成功")
	}

	if bytes.Equal(b[:len(B3)], B3) != true {
		t.Errorf("数据不匹配：%v", err)
	}

	if err := c.Close(); err != nil {
		t.Errorf("关闭连接错误。")
	}
}

func TestDirectProxyTCP(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:13336")
	if err != nil {
		t.Errorf("错误,%v", err)
	} else {
		fmt.Println("监听成功")
	}

	go testDirectProxyTCP1(t)

	b := make([]byte, 30)

	c, err := l.Accept()
	if err != nil {
		t.Errorf("接受连接错误：%v", err)
	} else {
		fmt.Println("接受链接成功")
	}

	if _, err := c.Write(B1); err != nil {
		t.Errorf("写数据失败：%v", err)
	} else {
		fmt.Println("写数据1成功")
	}

	if _, err := c.Read(b); err != nil {
		t.Errorf("读错误：%v", err)
	} else {
		fmt.Println("读b2成功")
	}
	if bytes.Equal(b[:len(B2)], B2) != true {
		t.Errorf("数据不匹配：%v", err)
	}

	if _, err := c.Write(B3); err != nil {
		t.Errorf("写数据失败：%v", err)
	} else {
		fmt.Println("写b3成功")
	}

	if _, err := c.Read(b); err != io.EOF {
		t.Errorf("读结尾错误：%v", err)
	}

}
