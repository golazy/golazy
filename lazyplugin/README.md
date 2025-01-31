# lazyplugin

## Description

The `lazyplugin` package provides a framework for creating and managing plugins in a Go application. It allows you to define plugins with specific functionality and integrate them into your application seamlessly.

## Usage

To use the `lazyplugin` package, follow these steps:

1. Import the package:

```go
import "golazy.dev/lazyplugin"
```

2. Define your plugin by implementing the `Plugin` interface:

```go
type MyPlugin struct{}

func (p *MyPlugin) Name() string {
    return "MyPlugin"
}

func (p *MyPlugin) Desc() string {
    return "This is my custom plugin."
}

func (p *MyPlugin) URL() string {
    return "https://example.com/myplugin"
}

func (p *MyPlugin) Init(ctx lazycontext.AppContext) {
    // Initialize the plugin
}
```

3. Register your plugin:

```go
lazyplugin.Plugins = append(lazyplugin.Plugins, &MyPlugin{})
```

4. Initialize all registered plugins in your application:

```go
func main() {
    ctx := lazycontext.NewAppContext()
    for _, plugin := range lazyplugin.Plugins {
        plugin.Init(ctx)
    }
    // Start your application
}
```

## Dependencies and Installation

To install the `lazyplugin` package, use the following command:

```sh
go get golazy.dev/lazyplugin
```

## Contributing and Reporting Issues

Contributions and issues are welcome. Please open an issue on the GitHub repository or submit a pull request with your changes.
