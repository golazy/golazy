package tcpdevserver

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func ExampleRunner() {

	var count int
	r := &Runner{
		Command: func() *exec.Cmd {
			count += 1
			cmd := exec.Command("bash", "-c", fmt.Sprintf(" echo %d ; sleep %d", count, count))
			cmd.Stdout = os.Stdout
			return cmd
		},
	}

	r.Start() // 1
	time.Sleep(time.Millisecond * 200)

	r.Restart() // 2
	time.Sleep(time.Millisecond * 200)

	r.Restart() // 3
	time.Sleep(time.Millisecond * 200)

	r.Restart() // 4
	time.Sleep(time.Millisecond * 200)

	r.Stop() // 4

	//r.Stop()

	// Output:
	// 1
	// 2
	// 3
	// 4

}
