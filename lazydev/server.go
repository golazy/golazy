package lazydev

import (
	"log"
	"net"
	"os"
	"reflect"
	"strconv"
	"syscall"
	"unsafe"
)

const (
	childEnvVariableName = "LAZYDEVCHILD"
)

var listener *net.TCPListener

func Serve() {
	log.SetPrefix("Parent(" + strconv.Itoa(os.Getpid()) + "):")
	log.SetFlags(0)
	setProcessName("ld: parent")

	if isParent := os.Getenv(childEnvVariableName) == ""; isParent {
		log.Println("I am the parent", os.Getpid())
		addr, err := net.ResolveTCPAddr("tcp", DefaultListenAddr)
		if err != nil {
			log.Fatal(err)
		}
		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			log.Fatal(err)
		}
		listener = l
		parentStart()
	} else {
		log.SetFlags(0)
		log.SetPrefix("Child (" + strconv.Itoa(os.Getpid()) + "):")
		log.Println("I am the child", os.Getpid())
		setProcessName("ld: child")
		childStart()
	}

	select {}
}

func setProcessName(name string) error {
	argv0str := (*reflect.StringHeader)(unsafe.Pointer(&os.Args[0]))
	argv0 := (*[1 << 30]byte)(unsafe.Pointer(argv0str.Data))[:argv0str.Len]

	n := copy(argv0, name)
	if n < len(argv0) {
		argv0[n] = 0
	}

	setProcessName2(name)
	return nil
}

func setProcessName2(name string) error {
	bytes := append([]byte(name), 0)
	ptr := unsafe.Pointer(&bytes[0])
	if _, _, errno := syscall.RawSyscall6(syscall.SYS_PRCTL, syscall.PR_SET_NAME, uintptr(ptr), 0, 0, 0, 0); errno != 0 {
		return syscall.Errno(errno)
	}
	return nil
}
