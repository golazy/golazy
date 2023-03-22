package commander


func TestCommander(t *testing.T) {

	commander := New()

	cmd := &Command{
		""

	}

	commander.RegisterJSCall(Command)
	commander.RegisterGoCall()
}