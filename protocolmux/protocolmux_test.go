package protocolmux

import (
	"fmt"
	"math/rand"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestProtocolMux(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	d := func(s string, args ...any) { t.Logf("T: "+s, args...) }
	//e := func(s string, args ...any) { t.Errorf("T: "+s, args...) }

	var wg sync.WaitGroup

	min := 2000
	max := 56000
	port := rand.Intn(max-min) + min
	addr := fmt.Sprintf(":%d", port)

	wg.Add(1)

	// Create the listener
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	// Create muxer
	mux := &Mux{L: l}
	holaListener := mux.ListenTo([][]byte{[]byte("hola")})

	// Start Listening
	go func() {
		defer wg.Done()
		t.Log("S: Listening in", addr)
		err := mux.Listen()
		if err != nil {
			t.Error(err)
			return
		}
	}()

	runtime.Gosched()

	wg.Add(1)
	// A client that writes "hola mundo" and expects "adios". Then clsoes
	go func() {
		d := func(s string, args ...any) { t.Logf("C: "+s, args...) }
		e := func(s string, args ...any) { t.Errorf("C: "+s, args...) }
		defer wg.Done()
		d("Dialing")
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Error(err)
		}
		d("Writing 'hola mundo'")
		for b := []byte("hola mundo"); len(b) > 0; {
			n, err := conn.Write(b)
			if err != nil {
				e("%v", err)
				return
			}
			d("Wrote %q", b[:n])
			b = b[n:]
		}

		b := make([]byte, 100)
		d("Waiting for reading")
		n, err := conn.Read(b)
		if err != nil {
			t.Error(err)
			return
		}
		d("Read: %q", string(b[:n]))
		if string(b[:n]) != "adios" {
			e("Expectin 'adios' got: %v", b[:n])
			return
		}

		err = conn.Close()
		if err != nil {
			e("%v", err)
			return
		}
	}()

	d("Waiting for client to holaListener connection")
	conn, err := holaListener.Accept()
	if err != nil {
		t.Fatal(err)
		return
	}
	d("Got a connection. Reading now")

	b := make([]byte, 50)
	n, err := conn.Read(b)
	if err != nil {
		t.Fatal(err)
		return
	}
	d("Read: %q", string(b[:n]))

	if string(b[:n]) != "hola mundo" {
		t.Fatal("Expecting 'hola mundo' got", string(b[:n]))
		return
	}

}
