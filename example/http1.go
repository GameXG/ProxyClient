package main
import (
	"github.com/gamexg/proxyclient"
	"fmt"
)

func main(){
	p,err:=proxyclient.NewProxyClient("http://127.0.0.1:7777")
	if err!=nil{
		panic(err)
	}

	c,err:=p.Dial("tcp","192.168.1.83:80")
	if err!=nil{
		panic(err)
	}

	if _,err:=c.Write([]byte("GET / HTTP/1.1\r\nHOST:127.0.0.1:6060\r\n\r\n"));err!=nil{
		panic(err)
	}

	b:=make([]byte,1024)

	if n,err:=c.Read(b);err!=nil{
		panic(err)
	}else{
		fmt.Print(string(b[:n]))
	}

}