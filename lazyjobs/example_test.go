package lazyjobs_test

import (
	"context"
	"fmt"
	"time"

	"golazy.dev/lazyjobs"
	"golazy.dev/lazyjobs/inmemoryjobs"
)

type welcomeEmailJob struct {
	lazyjobs.BaseJob
	Address string `json:"address"`
}

func (*welcomeEmailJob) Kind() string { return "mail.welcome" }

func (job *welcomeEmailJob) Work(context.Context) error {
	exampleSent <- job.Address
	return nil
}

var exampleSent chan string

func ExampleJobRunner() {
	exampleSent = make(chan string, 1)

	runner, err := lazyjobs.New(lazyjobs.Config{
		Backend: inmemoryjobs.New(),
		Define: func(runner *lazyjobs.JobRunner) {
			runner.MustRegister(&welcomeEmailJob{})
		},
		PollInterval: time.Millisecond,
	})
	if err != nil {
		panic(err)
	}
	defer runner.Stop(context.Background())

	if _, err := runner.Enqueue(context.Background(), &welcomeEmailJob{
		Address: "ada@example.com",
	}); err != nil {
		panic(err)
	}
	runner.Start(context.Background())

	select {
	case address := <-exampleSent:
		fmt.Println("sent", address)
	case <-time.After(time.Second):
		fmt.Println("job timed out")
	}

	// Output:
	// sent ada@example.com
}
