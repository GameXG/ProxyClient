package proxyclient

import (
	"net"

	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	SocksCmdConect       = 0x01
	SocksCmdBind         = 0x02
	SocksCmdUdpAssociate = 0x03
)

type SocksTCPConn struct {
	TCPConn
	localAddr, remoteAddr net.TCPAddr
	localHost, remoteHost string
	LocalPort, remotePort uint16
	proxyClient           ProxyClient
}

type SocksUDPConn struct {
	net.UDPConn
	proxyClient ProxyClient
}

type socksProxyClient struct {
	proxyAddr net.TCPAddr
	proxyType string // socks4 socks5
	//TODO: 用户名、密码
	upProxy ProxyClient
}

// 创建代理客户端
// ProxyType	socks4 socks5
// ProxyAddr 	127.0.0.1:5555
// UpProxy
func NewSocksProxyClient(proxyType string, proxyAddr string, upProxy ProxyClient) (ProxyClient, error) {
	nProxyType := strings.ToLower(strings.Trim(proxyType, " \r\n\t"))
	if nProxyType != "socks4" && nProxyType != "socks5" {
		return nil, errors.New("ProxyType 错误的格式")
	}

	nProxyAddr, err := net.ResolveTCPAddr("tcp", proxyAddr)
	if err != nil {
		return nil, errors.New("ProxyAddr 错误的格式")
	}

	if upProxy == nil {
		upProxy, err = NewDriectProxyClient("")
		if err != nil {
			return nil, fmt.Errorf("创建直连代理错误：%v", err)
		}
	}

	return &socksProxyClient{*nProxyAddr, nProxyType, upProxy}, nil
}

func (p *socksProxyClient) Dial(network, address string) (Conn, error) {
	if strings.HasPrefix(strings.ToLower(network), "tcp") {
		return p.DialTCPRAddr(network, address)
	} else if strings.HasPrefix(strings.ToLower(network), "udp") {
		addr, err := net.ResolveUDPAddr(network, address)
		if err != nil {
			return nil, fmt.Errorf("地址解析错误:%v", err)
		}
		return p.DialUDP(network, nil, addr)
	} else {
		return nil, errors.New("未知的 network 类型。")
	}
}

func (p *socksProxyClient) DialTimeout(network, address string, timeout time.Duration) (Conn, error) {
	return nil, errors.New("暂不支持")
}

func (p *socksProxyClient) DialTCP(network string, laddr, raddr *net.TCPAddr) (TCPConn, error) {
	if laddr != nil || laddr.Port != 0 {
		return nil, errors.New("代理协议不支持指定本地地址。")
	}

	return p.DialTCPRAddr(network, raddr.String())
}

func (p *socksProxyClient) DialTCPRAddr(network string, raddr string) (TCPConn, error) {
	c, err := p.upProxy.DialTCP(network, nil, &p.proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("无法连接代理服务器 %v ，错误：%v", p.proxyAddr, err)
	}

	if err := socksLogin(c, p); err != nil {
		return nil, fmt.Errorf("代理服务器登陆失败，错误：%v", err)
	}

	if err := socksSendCmdRequest(c, p, SocksCmdConect, raddr); err != nil {
		return nil, fmt.Errorf("请求代理服务器建立连接失败：%v", err)
	}

	_, _, _, _, _, cerr := socksRecvCmdResponse(c, p)
	if cerr != nil {
		return nil, fmt.Errorf("请求代理服务器建立连接失败：%v", err)
	}

	r := SocksTCPConn{TCPConn: c, proxyClient: p} //{c,net.ResolveTCPAddr("tcp","0.0.0.0:0"),net.ResolveTCPAddr("tcp","0.0.0.0:0"),"","",0,0  p}

	return &r, nil
}

func (p *socksProxyClient) DialUDP(network string, laddr, raddr *net.UDPAddr) (UDPConn, error) {
	return nil, errors.New("暂不支持 UDP 协议")
}

func (p *socksProxyClient) UpProxy() ProxyClient {
	return p.upProxy
}

func (p *socksProxyClient) SetUpProxy(upProxy ProxyClient) error {
	p.upProxy = upProxy
	return nil
}

func (c *SocksTCPConn) ProxyClient() ProxyClient {
	return c.proxyClient
}

func (c *SocksUDPConn) ProxyClient() ProxyClient {
	return c.proxyClient
}

// 登陆 socks 代理服务器
// 错误 err != nil ，不会关闭连接。
func socksLogin(c TCPConn, p *socksProxyClient) error {
	if p.proxyType == "socks4" || p.proxyType == "socks4a" {
		return nil
	} else if p.proxyType == "socks5" {
		//TODO: 目前只支持 无密码 鉴定，后期添加其他的鉴定支持。
		if _, err := c.Write([]byte{0x05, 0x01, 0x00}); err != nil {
			return fmt.Errorf("连接错误，发送数据失败。%v", err)
		}

		buf := make([]byte, 2)
		if _, err := io.ReadFull(c, buf); err != nil || bytes.Equal(buf, []byte{0x05, 0x00}) != true {
			return fmt.Errorf("服务器不支持“不需要鉴定”，回应：%v", buf)
		}
		return nil

	} else {
		return fmt.Errorf("不被支持的代理服务器类型: %v", p.proxyType)
	}
}

// 发送 socks 命令请求
func socksSendCmdRequest(w io.Writer, p *socksProxyClient, cmd byte, raddr string) error {
	b := make([]byte, 0, 6+len(raddr))

	var port uint16
	host, portString, err := net.SplitHostPort(raddr)
	if err != nil {
		return fmt.Errorf("raddr 格式错误。 raddr = %v", raddr)
	}

	hostSize := len(host)
	if hostSize > 255 && (p.proxyType == "socks5") {
		//TODO: 这里其实可以尝试本地解析域名，但是会造成实现不一致，还是不这么操作了。
		return fmt.Errorf("socks5 不支持超过255长度的域名，域名：%v", host)
	}

	if portUint64, err := strconv.ParseUint(portString, 10, 16); err != nil {
		return fmt.Errorf("port 超出有效范围。port=%v", portString)
	} else {
		port = uint16(portUint64)
	}

	portByte := make([]byte, 2)
	binary.BigEndian.PutUint16(portByte, port)

	// 如果 host 不是 ip 格式，则返回 nil
	ip := net.ParseIP(host)

	// socks4 不支持域名，所以需要本地解析域名
	if ip == nil && p.proxyType == "socks4" {
		ipaddr, err := net.ResolveIPAddr("ip4", host)
		if err != nil {
			return fmt.Errorf("[socks4] %v ipv4 解析失败，错误：%v", host, err)
		}
		ip = ipaddr.IP
	}

	// socks4a 使用 \0 作为域名结束符，所以需要检查 host 是否包含 \0
	if ip == nil && p.proxyType == "socks4a" && strings.Contains(host, string([]byte{0})) {
		return errors.New("[socks4a]错误的域名格式，域名内包含 \\0 ，和 socks4a 协议冲突，无法工作。")
	}

	// 将16字节长度的IPv4地址转换为4字节长
	if ip != nil && len(ip) != net.IPv4len {
		if nip := ip.To4(); nip != nil {
			ip = nip
		}
	}

	// socks4 、socks4a 只支持 IPv4 类型的IP
	if (p.proxyType == "socks4" || p.proxyType == "socks4a") && ip != nil && len(ip) != net.IPv4len {
		return fmt.Errorf("%v 协议不支持 IPv6 地址。域名：%v ,IP：%v", p.proxyType, host, ip.String())
	}

	if p.proxyType == "socks5" {
		if ip == nil { // 域名
			// Ver、CMD、RSV、 ATYP 、域名长度(前面解决了域名长度超过 byte 大小的问题)
			b = append(b, 0x05, cmd, 0x00, 0x03, byte(hostSize))
			// 域名
			b = append(b, []byte(host)...)
			// 端口
			b = append(b, portByte...)
		} else { //IPv4 or ipv6
			// Ver、CMD、RSV、ATYP
			if len(ip) == net.IPv4len {
				b = append(b, 0x05, cmd, 0x00, 0x01)
			} else if len(ip) == net.IPv6len {
				b = append(b, 0x05, cmd, 0x00, 0x04)
			} else {
				return errors.New("未知的IP格式。")
			}
			// ip
			b = append(b, []byte(ip)...)
			// 端口
			b = append(b, portByte...)
		}

	} else if ip == nil && p.proxyType == "socks4a" { // socks4a 域名格式
		// ver cmd
		b = append(b, 0x04, cmd)
		//port
		b = append(b, portByte...)
		// ip 、 -userid、null
		b = append(b, 0, 0, 0, 1, 0)
		//域名
		b = append(b, []byte(host)...)
		b = append(b, 0)

	} else if (ip != nil && p.proxyType == "socks4a") || p.proxyType == "socks4" { // 纯IP
		//ver cmd
		b = append(b, 0x04, cmd)
		//port
		b = append(b, portByte...)
		// ip
		b = append(b, []byte(ip)...)
	} else {
		return fmt.Errorf("未知的 socks 代理类型：%v", p.proxyType)
	}

	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("发送代理请求失败，%v", err)
	}

	return nil
}

// 接收代理服务器响应
// 服务器应答状态码成功时 err == nil
// 所以一般只需要判断 err 即可，不需要判断 rep
func socksRecvCmdResponse(r io.Reader, p *socksProxyClient) (rep int, dstAddr string, dstPort uint16, bndAddr string, bndPort uint16, err error) {
	b := make([]byte, 255+10)
	if p.proxyType == "socks4" || p.proxyType == "socks4a" {
		//ver
		if _, cerr := io.ReadFull(r, b[:1]); cerr != nil || b[0] != 0x04 {
			err = fmt.Errorf("socks4代理服务器 命令响应错误，ver=%v", b[0])
			return
		}

		// cmd、 dst port、dst ip
		if _, cerr := io.ReadFull(r, b[:7]); cerr != nil {
			err = fmt.Errorf("IO 读取错误，详细信息：%v", cerr)
			return
		}

		rep = int(b[0])
		if rep != 90 {
			err = fmt.Errorf("远程代理无法连接到目标。rep=%v", rep)
		}

		dstPort = binary.BigEndian.Uint16(b[1:3])
		dstIP := net.IP(b[3 : 3+4])
		dstAddr = dstIP.String()

		return
	} else if p.proxyType == "socks5" {
		//ver
		if _, cerr := io.ReadFull(r, b[:1]); cerr != nil || b[0] != 0x05 {
			err = fmt.Errorf("socks5代理服务器 命令响应错误，ver=%v", b[0])
			return
		}

		// rep rsv atyp domainSize
		if _, cerr := io.ReadFull(r, b[:4]); cerr != nil {
			err = fmt.Errorf("IO 读取错误，详细信息：%v", cerr)
			return
		}

		rep = int(b[0])
		atyp := b[2]
		domainSize := b[3]

		if rep != 0x00 {
			err = fmt.Errorf("远程代理无法连接到目标。rep=%v", b[0])
		}

		if atyp == 0x01 || atyp == 0x04 { //ipv4 or ipv6
			if atyp == 0x01 {
				b = b[:4]
			} else {
				b = b[:16]
			}

			b[0] = domainSize
			if _, cerr := io.ReadFull(r, b[1:]); cerr != nil {
				err = fmt.Errorf("IO 读取错误，详细信息：%v", cerr)
				return
			}
			bndAddr = net.IP(b).String()
		} else if atyp == 0x03 { //域名
			b = b[:domainSize]

			if _, cerr := io.ReadFull(r, b); cerr != nil {
				err = fmt.Errorf("IO 读取错误，详细信息：%v", cerr)
				return
			}
			bndAddr = string(b)
		} else { //未知的地址类型，不确定响应长度，退出
			err = fmt.Errorf("[socks5]未知的地址类型，atyp=%v", atyp)
			return
		}

		b = b[:2]
		if _, cerr := io.ReadFull(r, b); cerr != nil {
			err = fmt.Errorf("IO 读取错误，详细信息：%v", cerr)
			return
		}
		bndPort = binary.BigEndian.Uint16(b)
		return
	} else {
		err = fmt.Errorf("%v 不支持的协议类型", p.proxyType)
		return
	}
}
