package tcpdevserver

import (
	"fmt"
	"io"
	"net"
	"testing"
)

/*
func ExampleTcpDevServer() {

	s := &Server{
		Addr: ":9090",
		Child: func(l net.Listener) error {
			conn, _ := l.Accept()
			data, _ := io.ReadAll(conn)
			conn.Write(append(data, byte(0)))
			return nil
		},
	}

	done := make(chan (struct{}))
	s.Run(func(r Runner) error {
		r.Start()

		conn, _ := net.Dial("tcp", "localhost:9090")
		conn.Write(append([]byte("hola\n"), byte(0)))
		data, _ := io.ReadAll(conn)
		fmt.Println(string(data))

		r.Restart()
		conn, _ = net.Dial("tcp", "localhost:9090")
		conn.Write(append([]byte("mundo\n"), byte(0)))
		data, _ = io.ReadAll(conn)
		fmt.Println(string(data))

		done <- struct{}{}
		return nil
	})

	<-done
	// Output:
	// hola
	// mundo
}
*/

func TestTcpDevServer(t *testing.T) {

	s := &Server{
		Addr: ":9090",
		Child: func(l net.Listener) error {
			conn, err := l.Accept()
			if err != nil {
				panic(err)
			}
			data, err := io.ReadAll(conn)
			if err != nil {
				panic(err)
			}
			_, err = conn.Write(append(data, byte(0)))
			if err != nil {
				panic(err)
			}
			return nil
		},
	}

	done := make(chan (struct{}))
	s.Run(func(r *Runner) error {
		r.Start()

		conn, err := net.Dial("tcp", "localhost:9090")
		if err != nil {
			panic(err)
		}
		_, err = conn.Write(append([]byte("hola\n"), byte(0)))
		if err != nil {
			panic(err)
		}
		data, err := io.ReadAll(conn)
		if err != nil {
			panic(err)
		}
		if string(data) != "hola\n" {
			t.Fatal("Expecting hola. Got", string(data))
		}

		r.Restart()

		conn, _ = net.Dial("tcp", "localhost:9090")
		conn.Write(append([]byte("mundo\n"), byte(0)))
		data, _ = io.ReadAll(conn)
		fmt.Println(string(data))

		done <- struct{}{}
		return nil
	})

	<-done
	// Output:
	// hola
	// mundo
}
