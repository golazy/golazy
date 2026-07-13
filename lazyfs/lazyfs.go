package lazyfs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"
	"sync"
)

// ErrSealed is returned when a layer is added to a sealed filesystem.
var ErrSealed = errors.New("lazyfs: filesystem is sealed")

// Layer describes one filesystem in a layered stack. Index zero is the lowest
// precedence layer.
type Layer struct {
	Index int
	Name  string
	Owner string
}

// Candidate describes a layer containing a resolved path.
type Candidate struct {
	Layer Layer
	Mode  fs.FileMode
	IsDir bool
}

// Resolution describes which layer wins for a path and which lower layers are
// shadowed. Candidates are ordered from highest to lowest precedence, so the
// first candidate is also Winner.
type Resolution struct {
	Path       string
	Winner     Candidate
	Candidates []Candidate
}

// AddOption configures metadata for an added layer.
type AddOption interface {
	apply(*addOptions)
}

type addOption func(*addOptions)

func (option addOption) apply(options *addOptions) {
	option(options)
}

type addOptions struct {
	name  string
	owner string
}

// Name assigns a stable, human-readable name to a layer. Non-empty names must
// be unique within one filesystem.
func Name(name string) AddOption {
	return addOption(func(options *addOptions) {
		options.name = strings.TrimSpace(name)
	})
}

// Owner records the package or add-on that contributed a layer.
func Owner(owner string) AddOption {
	return addOption(func(options *addOptions) {
		options.owner = strings.TrimSpace(owner)
	})
}

type layer struct {
	metadata Layer
	files    fs.FS
}

// FS is a concurrency-safe layered filesystem. The last filesystem added has
// the highest precedence.
type FS struct {
	mu     sync.RWMutex
	layers []layer
	sealed bool
}

// New returns an empty layered filesystem.
func New() *FS {
	return &FS{}
}

// Add adds files as the new highest-precedence layer.
func (files *FS) Add(fsys fs.FS, options ...AddOption) error {
	if files == nil {
		return fmt.Errorf("lazyfs: add to nil filesystem")
	}
	if fsys == nil {
		return fmt.Errorf("lazyfs: added filesystem is nil")
	}
	configured := addOptions{}
	for _, option := range options {
		if option != nil {
			option.apply(&configured)
		}
	}

	files.mu.Lock()
	defer files.mu.Unlock()
	if files.sealed {
		return ErrSealed
	}
	if configured.name != "" {
		for _, existing := range files.layers {
			if existing.metadata.Name == configured.name {
				return fmt.Errorf("lazyfs: layer name %q is already registered", configured.name)
			}
		}
	}
	files.layers = append(files.layers, layer{
		metadata: Layer{
			Index: len(files.layers),
			Name:  configured.name,
			Owner: configured.owner,
		},
		files: fsys,
	})
	return nil
}

// Seal prevents later additions. Calling Seal more than once is safe.
func (files *FS) Seal() error {
	if files == nil {
		return fmt.Errorf("lazyfs: seal nil filesystem")
	}
	files.mu.Lock()
	files.sealed = true
	files.mu.Unlock()
	return nil
}

// Sealed reports whether Seal has been called.
func (files *FS) Sealed() bool {
	if files == nil {
		return false
	}
	files.mu.RLock()
	defer files.mu.RUnlock()
	return files.sealed
}

// Layers returns layer metadata ordered from lowest to highest precedence.
func (files *FS) Layers() []Layer {
	layers := files.snapshot()
	result := make([]Layer, len(layers))
	for index, item := range layers {
		result[index] = item.metadata
	}
	return result
}

// Resolve reports the winning and shadowed layers for name.
func (files *FS) Resolve(name string) (Resolution, error) {
	if err := validPath("resolve", name); err != nil {
		return Resolution{}, err
	}
	layers := files.snapshot()
	candidates := make([]Candidate, 0, len(layers))
	for index := len(layers) - 1; index >= 0; index-- {
		info, exists, blocked, err := lookupInLayer(layers[index], name, false)
		if err != nil {
			return Resolution{}, err
		}
		if blocked {
			if len(candidates) == 0 {
				return Resolution{}, pathError("resolve", name, fs.ErrNotExist)
			}
			break
		}
		if !exists {
			continue
		}
		candidates = append(candidates, Candidate{
			Layer: layers[index].metadata,
			Mode:  info.Mode(),
			IsDir: info.IsDir(),
		})
	}
	if len(candidates) == 0 {
		return Resolution{}, pathError("resolve", name, fs.ErrNotExist)
	}
	return Resolution{
		Path:       name,
		Winner:     candidates[0],
		Candidates: candidates,
	}, nil
}

// Open implements fs.FS.
func (files *FS) Open(name string) (fs.File, error) {
	if err := validPath("open", name); err != nil {
		return nil, err
	}
	layers := files.snapshot()
	index, info, err := winnerIn(layers, name, false)
	if err != nil {
		return nil, err
	}
	opened, err := layers[index].files.Open(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return opened, nil
	}
	_ = opened.Close()
	entries, err := readMergedDir(layers, index, name)
	if err != nil {
		return nil, err
	}
	return &directory{info: info, entries: entries}, nil
}

// ReadFile implements fs.ReadFileFS.
func (files *FS) ReadFile(name string) ([]byte, error) {
	if err := validPath("read", name); err != nil {
		return nil, err
	}
	layer, _, err := files.winner(name, false)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(layer.files, name)
}

// ReadDir implements fs.ReadDirFS.
func (files *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	if err := validPath("readdir", name); err != nil {
		return nil, err
	}
	layers := files.snapshot()
	index, info, err := winnerIn(layers, name, false)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return fs.ReadDir(layers[index].files, name)
	}
	return readMergedDir(layers, index, name)
}

// Stat implements fs.StatFS.
func (files *FS) Stat(name string) (fs.FileInfo, error) {
	if err := validPath("stat", name); err != nil {
		return nil, err
	}
	_, info, err := files.winner(name, false)
	return info, err
}

// Glob implements fs.GlobFS.
func (files *FS) Glob(pattern string) ([]string, error) {
	return fs.Glob(openOnly{FS: files}, pattern)
}

// Sub implements fs.SubFS. The returned filesystem is a sealed snapshot that
// preserves the source layer metadata.
func (files *FS) Sub(dir string) (fs.FS, error) {
	if err := validPath("sub", dir); err != nil {
		return nil, err
	}
	if dir == "." {
		return files, nil
	}
	layers := files.snapshot()
	index, info, err := winnerIn(layers, dir, false)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, pathError("sub", dir, fmt.Errorf("%w: not a directory", fs.ErrInvalid))
	}

	result := New()
	directoryLayers, err := visibleDirectoryLayers(layers, index, dir)
	if err != nil {
		return nil, err
	}
	for _, item := range directoryLayers {
		sub, err := fs.Sub(item.files, dir)
		if err != nil {
			return nil, err
		}
		if err := result.Add(sub, Name(item.metadata.Name), Owner(item.metadata.Owner)); err != nil {
			return nil, err
		}
	}
	_ = result.Seal()
	return result, nil
}

// ReadLink implements fs.ReadLinkFS.
func (files *FS) ReadLink(name string) (string, error) {
	if err := validPath("readlink", name); err != nil {
		return "", err
	}
	layer, _, err := files.winner(name, true)
	if err != nil {
		return "", err
	}
	return fs.ReadLink(layer.files, name)
}

// Lstat implements fs.ReadLinkFS.
func (files *FS) Lstat(name string) (fs.FileInfo, error) {
	if err := validPath("lstat", name); err != nil {
		return nil, err
	}
	_, info, err := files.winner(name, true)
	return info, err
}

func (files *FS) snapshot() []layer {
	if files == nil {
		return nil
	}
	files.mu.RLock()
	defer files.mu.RUnlock()
	return append([]layer(nil), files.layers...)
}

func (files *FS) winner(name string, lstat bool) (layer, fs.FileInfo, error) {
	layers := files.snapshot()
	index, info, err := winnerIn(layers, name, lstat)
	if err != nil {
		return layer{}, nil, err
	}
	return layers[index], info, nil
}

func winnerIn(layers []layer, name string, lstat bool) (int, fs.FileInfo, error) {
	for index := len(layers) - 1; index >= 0; index-- {
		info, exists, blocked, err := lookupInLayer(layers[index], name, lstat)
		if err != nil {
			return -1, nil, err
		}
		if blocked {
			return -1, nil, pathError("stat", name, fs.ErrNotExist)
		}
		if exists {
			return index, info, nil
		}
	}
	return -1, nil, pathError("stat", name, fs.ErrNotExist)
}

// lookupInLayer distinguishes an absent path from a path hidden by an entry in
// the same layer. A non-directory ancestor (or a broken symlink at the target)
// is a barrier: lower-precedence layers must not be consulted.
func lookupInLayer(item layer, name string, lstat bool) (fs.FileInfo, bool, bool, error) {
	blocked, err := blockedByAncestor(item.files, name)
	if err != nil {
		return nil, false, false, err
	}
	if blocked {
		return nil, false, true, nil
	}

	var info fs.FileInfo
	if lstat {
		info, err = fs.Lstat(item.files, name)
	} else {
		info, err = fs.Stat(item.files, name)
	}
	if err == nil {
		return info, true, false, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, false, false, err
	}
	if lstat {
		return nil, false, false, nil
	}

	// Stat follows links. If Lstat still finds an entry, the target is a
	// broken link and shadows any lower path of the same name.
	if _, lstatErr := fs.Lstat(item.files, name); lstatErr == nil {
		return nil, false, true, nil
	} else if !errors.Is(lstatErr, fs.ErrNotExist) {
		return nil, false, false, lstatErr
	}
	return nil, false, false, nil
}

func blockedByAncestor(files fs.FS, name string) (bool, error) {
	for index := 0; index < len(name); index++ {
		if name[index] != '/' {
			continue
		}
		ancestor := name[:index]
		info, err := fs.Stat(files, ancestor)
		if err == nil {
			if !info.IsDir() {
				return true, nil
			}
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return false, err
		}

		// A broken link is not visible through Stat, but it still blocks
		// traversal into a lower layer.
		info, err = fs.Lstat(files, ancestor)
		if err == nil {
			if !info.IsDir() {
				return true, nil
			}
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return false, err
		}
	}
	return false, nil
}

// visibleDirectoryLayers returns directory layers from lowest to highest
// precedence. Scanning stops at the first lower file barrier, so a high
// directory never merges descendants from beneath a shadowing file.
func visibleDirectoryLayers(layers []layer, winner int, name string) ([]layer, error) {
	directories := make([]layer, 0, winner+1)
	for index := winner; index >= 0; index-- {
		info, exists, blocked, err := lookupInLayer(layers[index], name, false)
		if err != nil {
			return nil, err
		}
		if blocked {
			break
		}
		if !exists {
			continue
		}
		if !info.IsDir() {
			break
		}
		directories = append(directories, layers[index])
	}
	for left, right := 0, len(directories)-1; left < right; left, right = left+1, right-1 {
		directories[left], directories[right] = directories[right], directories[left]
	}
	return directories, nil
}

func readMergedDir(layers []layer, winner int, name string) ([]fs.DirEntry, error) {
	directoryLayers, err := visibleDirectoryLayers(layers, winner, name)
	if err != nil {
		return nil, err
	}
	entriesByName := map[string]fs.DirEntry{}
	for _, item := range directoryLayers {
		entries, err := fs.ReadDir(item.files, name)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			entriesByName[entry.Name()] = entry
		}
	}

	names := make([]string, 0, len(entriesByName))
	for name := range entriesByName {
		names = append(names, name)
	}
	sort.Strings(names)
	entries := make([]fs.DirEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, entriesByName[name])
	}
	return entries, nil
}

func validPath(operation string, name string) error {
	if fs.ValidPath(name) {
		return nil
	}
	return pathError(operation, name, fs.ErrInvalid)
}

func pathError(operation string, name string, err error) error {
	return &fs.PathError{Op: operation, Path: name, Err: err}
}

type openOnly struct {
	fs.FS
}

type directory struct {
	info    fs.FileInfo
	entries []fs.DirEntry
	offset  int
}

func (dir *directory) Stat() (fs.FileInfo, error) {
	return dir.info, nil
}

func (dir *directory) Read([]byte) (int, error) {
	return 0, pathError("read", dir.info.Name(), fs.ErrInvalid)
}

func (dir *directory) Close() error {
	return nil
}

func (dir *directory) ReadDir(count int) ([]fs.DirEntry, error) {
	if dir.offset >= len(dir.entries) && count > 0 {
		return nil, io.EOF
	}
	if count <= 0 {
		remaining := dir.entries[dir.offset:]
		dir.offset = len(dir.entries)
		return remaining, nil
	}
	end := min(dir.offset+count, len(dir.entries))
	entries := dir.entries[dir.offset:end]
	dir.offset = end
	if len(entries) == 0 {
		return nil, io.EOF
	}
	return entries, nil
}

var (
	_ fs.FS         = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
	_ fs.GlobFS     = (*FS)(nil)
	_ fs.SubFS      = (*FS)(nil)
	_ fs.ReadLinkFS = (*FS)(nil)
)
