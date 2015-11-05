package proxyclient

import (
	"testing"
	"bytes"
	"io"
)

func TestByteReader(t*testing.T) {
	r := ByteReader{[][]byte{[]byte{0, 1, 2}, []byte{3, 4}, []byte{5}, []byte{6, 7, 8, 9}}, 0}
	b := make([]byte, 10)

	f := func(tb []byte) () {
		if n, err := r.Read(b); err != nil || n != len(tb) || bytes.Equal(tb, b[:n]) != true {
			t.Errorf("读错误")
		}
		return
	}

	f([]byte{0, 1, 2})
	f([]byte{3, 4})
	f([]byte{5})
	f([]byte{6, 7, 8, 9})
	f([]byte{0})

	if n, err := r.Read(b); n != 0 || err != io.EOF {
		t.Errorf("读结尾错误")
	}
}


func Test_readData(t *testing.T) {
	r := ByteReader{[][]byte{[]byte{0, 1, 2}, []byte{3, 4}, []byte{5}, []byte{6, 7, 8, 9}}, 0}

	b := make([]byte, 4)
	if n, err := ReadData(r, b); n != 4 || err != nil || bytes.Equal(b, []byte{0, 1, 2, 3}) {
		t.Error("读错误")
	}

	b = make([]byte, 20)
	if n, err := ReadData(r, b); n != 5 || err != nil || bytes.Equal(b[:5], []byte{5, 6, 7, 8, 9}) {
		t.Error("读错误")
	}

	if n, err := ReadData(r, b); n != 0 || err != io.EOF {
		t.Errorf("读结尾错误")
	}
}








