# lazyenv

## Description

lazyenv is a package that provides easy access to environment variables and simplifies the process of filling struct fields with environment variable values. It offers functions to retrieve environment variables, convert them to the appropriate types, and fill struct fields based on their names or tags.

## Usage

### Retrieving Environment Variables

You can use the `Get` function to retrieve the value of an environment variable and convert it to the desired type. Here's an example:

```go
package main

import (
	"fmt"
	"golazy.dev/lazyenv"
)

func main() {
	os.Setenv("DB_NAME", "test_db")
	dbName := lazyenv.Get[string]("DB_NAME")
	fmt.Println("DB Name:", dbName)
}
```

### Filling Struct Fields

You can use the `Fill` function to fill the fields of a struct with the values from environment variables. The function will use the uppercase and dash-separated name of the field as the environment variable name. You can also override the environment variable name using the `env` tag. Here's an example:

```go
package main

import (
	"fmt"
	"golazy.dev/lazyenv"
)

type Config struct {
	DB struct {
		Name     string
		Host     string
		Password string `env:"PASS"`
	}
	UserID     int
	HTTPServer string
	Simple     bool
	Another    []int
	MapExample map[string]string
	LowerCase  string `env:"lowercase"`
}

func main() {
	os.Setenv("DB_NAME", "test_db")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PASS", "localhost")
	os.Setenv("USER_ID", "42")
	os.Setenv("HTTP_SERVER", "localhost")
	os.Setenv("SIMPLE", "true")
	os.Setenv("ANOTHER", "1,2,3")
	os.Setenv("MAP_EXAMPLE", "key1:value1,key2:value2")
	os.Setenv("lowercase", "with_env")

	var config Config
	lazyenv.Fill(&config)

	fmt.Printf("%+v\n", config)
}
```

### Using Default Values

You can use the `GetDefault` function to retrieve the value of an environment variable and provide a default value if the environment variable is not set. Here's an example:

```go
package main

import (
	"fmt"
	"golazy.dev/lazyenv"
)

func main() {
	os.Setenv("DB_NAME", "test_db")
	dbName := lazyenv.GetDefault("DB_NAME", "default_db")
	fmt.Println("DB Name:", dbName)

	missingVar := lazyenv.GetDefault("MISSING_VAR", "default_value")
	fmt.Println("Missing Var:", missingVar)
}
```

## Dependencies and Installation

To use lazyenv, you need to have Go installed on your system. You can install lazyenv using the following command:

```sh
go get golazy.dev/lazyenv
```

## Contributing and Reporting Issues

If you would like to contribute to the development of lazyenv or report any issues, please visit the [GitHub repository](https://github.com/golazy/golazy) and follow the guidelines for contributing and reporting issues.
