# lazyassets

## Description

The `lazyassets` package provides a simple and efficient way to manage static assets in a Go web application. It allows you to serve static files with cache busting and content hashing, ensuring that your users always receive the most up-to-date version of your assets.

## Usage

To use the `lazyassets` package, follow these steps:

1. Import the package:

```go
import "github.com/golazy/golazy/lazyassets"
```

2. Create a new `Server` instance:

```go
server := &lazyassets.Server{
    Storage: &lazyassets.Storage{},
}
```

3. Add your static files to the server:

```go
server.Add("/path/to/your/file.css", "file content")
```

4. Serve the static files in your HTTP handler:

```go
http.Handle("/assets/", http.StripPrefix("/assets/", server))
```

## Dependencies and Installation

The `lazyassets` package has no external dependencies. To install the package, run the following command:

```sh
go get github.com/golazy/golazy/lazyassets
```

## Contributing and Reporting Issues

If you would like to contribute to the `lazyassets` package or report any issues, please visit the [GitHub repository](https://github.com/golazy/golazy) and follow the guidelines for contributing and reporting issues.
