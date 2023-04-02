package project

import "testing"

type Vars map[string]string

func TestApp(t *testing.T) {

	App.Generate("app_test", Vars{"App": "Juan"})

}
