package proxyclient
import "io"

// 读数据
// 除非遇到错误，否则会读满 b 。
// err == nil 时 n == len(b)
func ReadData(r io.Reader, b []byte) (n int, err error) {
	//TODO: 坑死，这才发现标准库里面有这个函数 io.ReafFull 。
	for n < len(b) {
		cn, cerr := r.Read(b[n:])
		n += cn
		if cerr != nil {
			return n, cerr
		}
	}
	return n, nil
}

// [][]byte Reader ，和标准库的区别是可以定义每次读取时的内容
type ByteReader struct {
	data [][]byte
	o    int
}

func (r *ByteReader)Read(b []byte) (n int, err error) {
	if r.o >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(r.data[r.o], b)
	r.o += 1
	return n, nil
}
