# LazySupport

lazysupport is a module of golazy that provides various utility functions and helpers to support other golazy modules.

## Description

lazysupport is a package that offers a collection of utility functions and helpers that can be used across different golazy modules. It includes functions for caching, DOM manipulation, string manipulation, and more.

## Usage

### Caching

The `lazysupport` package provides a simple in-memory cache that can be used to cache the results of expensive operations. Here's an example:

```go
package main

import (
	"fmt"
	"golazy.dev/lazysupport"
)

func main() {
	cache := lazysupport.MemCache{}
	result, err := cache.Cache(expensiveOperation, "key")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Result:", string(result))
}

func expensiveOperation() ([]byte, error) {
	// Perform an expensive operation
	return []byte("expensive result"), nil
}
```

### DOM Manipulation

The `lazysupport` package provides functions for generating DOM IDs based on struct values. Here's an example:

```go
package main

import (
	"fmt"
	"golazy.dev/lazysupport"
)

type User struct {
	ID uint
}

func main() {
	user := User{ID: 1}
	domID, err := lazysupport.DomID(user)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("DOM ID:", domID)
}
```

## Dependencies and Installation

To use lazysupport, you need to have Go installed on your system. You can install lazysupport using the following command:

```sh
go get golazy.dev/lazysupport
```

## Contributing and Reporting Issues

If you would like to contribute to the development of lazysupport or report any issues, please visit the [GitHub repository](https://github.com/golazy/golazy) and follow the guidelines for contributing and reporting issues.
