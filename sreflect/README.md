# sreflect

## Description

The `sreflect` package helps in describing Go structs, their methods, fields, and embedded fields. It provides a way to reflect on the structure of Go types and retrieve detailed information about them.

## Usage

### Reflecting on a Struct

To reflect on a struct using the `sreflect` package, you need to create a new instance of the struct and pass it to the `Reflect` function. Here's an example:

```go
package main

import (
	"fmt"
	"sreflect"
)

type MyStruct struct {
	Name string
}

func main() {
	s := &MyStruct{Name: "example"}
	info := sreflect.Reflect(s)
	fmt.Println("Struct Name:", info.Name())
}
```

### Accessing Methods

You can access the methods of a struct and its embedded structs using the `AllMethods` function. Here's an example:

```go
package main

import (
	"fmt"
	"sreflect"
)

type MyStruct struct {
	Name string
}

func (m *MyStruct) Greet() {
	fmt.Println("Hello,", m.Name)
}

func main() {
	s := &MyStruct{Name: "example"}
	info := sreflect.Reflect(s)
	methods := info.AllMethods()
	for _, method := range methods {
		fmt.Println("Method Name:", method.Name)
	}
}
```

### Calling Methods

You can call methods on a struct using the `Call` function. Here's an example:

```go
package main

import (
	"fmt"
	"sreflect"
)

type MyStruct struct {
	Name string
}

func (m *MyStruct) Greet(greeting string) {
	fmt.Println(greeting, m.Name)
}

func main() {
	s := &MyStruct{Name: "example"}
	info := sreflect.Reflect(s)
	methods := info.AllMethods()
	for _, method := range methods {
		if method.Name == "Greet" {
			resolver := func(t reflect.Type) (reflect.Value, error) {
				if t.Kind() == reflect.String {
					return reflect.ValueOf("Hello"), nil
				}
				return reflect.Value{}, fmt.Errorf("unsupported type: %s", t)
			}
			method.Call(reflect.ValueOf(s), resolver)
		}
	}
}
```

## Dependencies and Installation

To use the `sreflect` package, you need to have Go installed on your system. You can install the `sreflect` package using the following command:

```sh
go get golazy.dev/sreflect
```

## Contributing and Reporting Issues

If you would like to contribute to the development of the `sreflect` package or report any issues, please visit the [GitHub repository](https://github.com/golazy/golazy) and follow the guidelines for contributing and reporting issues.
