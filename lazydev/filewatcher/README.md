# filewatcher

Package filewatcher notifies when the filesystem has change.
It goes up to the top directory that holds a go.mod file

## Variables

IgnoredDirs is a list of directories that should not tirgger a change

```golang
var IgnoredDirs = []string{".git", "log"}
```

IgnoredFiles is a list of files that should not trigger a change

```golang
var IgnoredFiles = []string{}
```

## Types

### type [Change](/filewatcher.go#L27)

`type Change struct { ... }`

Change represent a change in the filesystem

### type [ChangeSet](/filewatcher.go#L33)

`type ChangeSet []Change`

ChangeSet is a collection of changes

### type [FileWatcher](/filewatcher.go#L36)

`type FileWatcher struct { ... }`

FileWatcher looks for changes in the top most directory that have a go.mod

#### func [New](/filewatcher.go#L45)

`func New(dir string) (fw *FileWatcher, err error)`

New initializes a FileWatcher in the given directory
It will go up to the top most directory that holds a go.mod
If dir is an empty string it will use the current directory

#### func (*FileWatcher) [Close](/filewatcher.go#L68)

`func (fw *FileWatcher) Close() error`

Close stop listening for changes in the file system
Once close, the channel will be closed

#### func (*FileWatcher) [Watch](/filewatcher.go#L73)

`func (fw *FileWatcher) Watch() (<-chan (ChangeSet), error)`

Watch start watching for recursively in the project

### type [Op](/filewatcher.go#L19)

`type Op fsnotify.Op`

Op holds the operation name

#### func (Op) [String](/filewatcher.go#L22)

`func (o Op) String() string`

String return the operation name Create , Write , Remove , Rename or Chmod

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
