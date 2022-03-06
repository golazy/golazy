package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/guillermo/golazy/lazydev/tcpdevserver"
)

func EchoServer(l net.Listener) error {

	log := func(args ...interface{}) { fmt.Println(append([]interface{}{"EchoServer: "}, args...)...) }
	for {
		log("Waiting for connection")
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		log("Got connection")
		for {

			log("Waiting for data")
			b := make([]byte, 1024)
			n, err := conn.Read(b)
			if err != nil {
				log("Error while reading", err)
				break
			}
			log("Writting data")
			_, err = conn.Write(b[:n])
			if err != nil {
				log("Error while writting", err)
				break
			}
		}
		log("Clossing connection")
		conn.Close()
	}
}

func Client() {
	log := func(args ...interface{}) { fmt.Println(append([]interface{}{"client: "}, args...)...) }
	time.Sleep(1)
	for {
		log("Dialing")
		conn, err := net.Dial("tcp", "localhost:9090")
		if err != nil {
			panic(err)
		}
		log("Writting data")
		_, err = conn.Write(append([]byte("hola\n"), byte(0)))
		if err != nil {
			panic(err)
		}
		log("Waiting for data")
		data, err := io.ReadAll(conn)
		if err != nil {
			panic(err)
		}
		log("Got data", string(data), "clossing connection")
		if string(data) != "hola\n" {
			log(fmt.Errorf("expecting hola. Got: %q", string(data)))
		}
		err = conn.Close()
		if err != nil {
			panic(err)
		}

		fmt.Println("Sleeping")
	}
}

func main() {
	log.SetFlags(0)
	server := &tcpdevserver.Server{
		Addr: ":9090",
		Log:  log.Default(),
	}

	server.Child = EchoServer

	//server.AfterListen = Client

	server.Run(func(r *tcpdevserver.Runner) error {
		fmt.Println("--------------------------------------------")
		r.Start()
		for {
			time.Sleep(time.Second * 10)
			fmt.Println("--------------------------------------------")
			r.Restart()
		}

		return nil
	})

}
