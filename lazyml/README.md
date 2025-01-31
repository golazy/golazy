# lazyml

## Description

lazyml is a package that provides a flexible and efficient way to create and manipulate HTML elements in Go. It allows you to build HTML documents programmatically with a clean and organized structure.

## Usage

### Creating Elements

To create elements using lazyml, you need to use the `NewElement` function and provide the tag name and any attributes or children. Here's an example:

```go
package main

import (
	"fmt"
	"golazy.dev/lazyml"
)

func main() {
	element := lazyml.NewElement("div", lazyml.NewAttr("class", "container"), "Hello, World!")
	fmt.Println(element.String())
}
```

### Using Attributes

You can define attributes for elements using the `NewAttr` function. Here's an example of creating an element with multiple attributes:

```go
package main

import (
	"fmt"
	"golazy.dev/lazyml"
)

func main() {
	element := lazyml.NewElement("a", lazyml.NewAttr("href", "https://example.com"), lazyml.NewAttr("target", "_blank"), "Click here")
	fmt.Println(element.String())
}
```

### Nesting Elements

You can nest elements by passing other elements as children. Here's an example of creating a nested structure:

```go
package main

import (
	"fmt"
	"golazy.dev/lazyml"
)

func main() {
	child := lazyml.NewElement("span", "Nested element")
	parent := lazyml.NewElement("div", lazyml.NewAttr("class", "parent"), child)
	fmt.Println(parent.String())
}
```

## Dependencies and Installation

To use lazyml, you need to have Go installed on your system. You can install lazyml using the following command:

```sh
go get golazy.dev/lazyml
```

## Contributing and Reporting Issues

If you would like to contribute to the development of lazyml or report any issues, please visit the [GitHub repository](https://github.com/golazy/golazy) and follow the guidelines for contributing and reporting issues.
