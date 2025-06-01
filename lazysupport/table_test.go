package lazysupport

import (
	"fmt"
)

func ExampleTable() {

	table := Table{
		Header: []string{"Title", "Age"},
		Values: [][]string{
			{"The film", "1"},
			{"The super super file"},
			{"", ""},
			{"Other Filem", "123123"},
		},
	}

	fmt.Println(table.String())

	// Output:
	// Title                Age
	// The film             1
	// The super super file
	//
	// Other Filem          123123

}
