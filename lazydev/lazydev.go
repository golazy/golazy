/* Package lazydev implements an autoreload server

The main method Serve() will start a child process with `go run *.go` (except
test files). everytime there is a change in the filesystem, it will stop the
server and start again.


The default port can be changed through the PORT environment variable or through DefaultListenAddr

  lazydev.DefaultListenAddr = ":9090"

By default it uses http.DefaultServeMux but can be changed through DefaultServerMux

  lazydev.DefaultServeMux = http.HandlerFunc(func(w http.RespnoseWriter, r *http.Request){w.Write([]byte("hello"))})

It watches for changes in WatchPaths that defaults to the current directory. More can be added by modifing the variable or by the LAZYWATCH environment variable.

	LAZYWATCH="./..." go run *.go

*/
package lazydev
