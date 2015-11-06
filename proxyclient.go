package proxyclient

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// 连接
type Conn interface {
	net.Conn
	ProxyClient() ProxyClient // 获得所属的代理
}

// 表示 TCP 连接
type TCPConn interface {
	Conn

	/*
		SetLinger设定当连接中仍有数据等待发送或接受时的Close方法的行为。
		如果sec < 0（默认），Close方法立即返回，操作系统停止后台数据发送；如果 sec == 0，Close立刻返回，操作系统丢弃任何未发送或未接收的数据；如果sec > 0，Close方法阻塞最多sec秒，等待数据发送或者接收，在一些操作系统中，在超时后，任何未发送的数据会被丢弃。
	*/
	SetLinger(sec int) error

	// SetNoDelay设定操作系统是否应该延迟数据包传递，以便发送更少的数据包（Nagle's算法）。默认为真，即数据应该在Write方法后立刻发送。
	SetNoDelay(noDelay bool) error

	//SetReadBuffer设置该连接的系统接收缓冲
	SetReadBuffer(bytes int) error

	//SetWriteBuffer设置该连接的系统发送缓冲
	SetWriteBuffer(bytes int) error
}

// 表示 UDP 连接
type UDPConn interface {
	Conn
}

// 仿 net 库接口
type ProxyClient interface {
	UpProxy() ProxyClient
	SetUpProxy(upProxy ProxyClient) error

	Dial(network, address string) (Conn, error)
	DialTimeout(network, address string, timeout time.Duration) (Conn, error)
	DialTCP(net string, laddr, raddr *net.TCPAddr) (TCPConn, error)
	//ListenTCP在本地TCP地址laddr上声明并返回一个*TCPListener，net参数必须是"tcp"、"tcp4"、"tcp6"，如果laddr的端口字段为0，函数将选择一个当前可用的端口，可以用Listener的Addr方法获得该端口。
	//ListenTCP(net string, laddr *TCPAddr) (*TCPListener, error)
	//DialTCP在网络协议net上连接本地地址laddr和远端地址raddr。net必须是"udp"、"udp4"、"udp6"；如果laddr不是nil，将使用它作为本地地址，否则自动选择一个本地地址。
	DialUDP(net string, laddr, raddr *net.UDPAddr) (UDPConn, error)
}

// 创建代理客户端
// http 代理 http://123.123.123.123:8088
// socks4 代理 socks4://123.123.123.123:5050  不支持远端 dns 解析
// socks4a 代理 socks4a://123.123.123.123:5050
// socks5 代理 socks5://123.123.123.123:5050?upProxy=http://145.2.1.3:8080
// 直连 direct://0.0.0.0:0000/?LocalAddr=123.123.123.123:0
func NewProxyClient(addr string) (ProxyClient, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, errors.New("addr 错误的格式")
	}
	_query, err := url.ParseQuery(u.RawQuery)
	query := make(map[string][]string)
	for k, v := range _query {
		query[strings.ToLower(k)] = v
	}

	scheme := strings.ToLower(strings.TrimSpace(u.Scheme))

	var upProxy ProxyClient = nil
	if up, ok := query["upproxy"]; ok == true {
		if upProxy, err = NewDriectProxyClient(up[0]); err != nil {
			return nil, fmt.Errorf("upProxy 创建失败：%v", err)
		}
	}

	switch scheme {
	case "direct":
		if localAddr, ok := query["LocalAddr"]; ok {
			return NewDriectProxyClient(localAddr[0])
		} else {
			return NewDriectProxyClient(":0")
		}
	case "socks4", "socks4a", "socks5":
		return NewSocksProxyClient(scheme, u.Host, upProxy)
	default:
		return nil, fmt.Errorf("未识别的代理类型：%v", scheme)
	}
}
