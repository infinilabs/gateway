package main

import (
	"bytes"
	"github.com/gorilla/context"
	httprouter "infini.sh/framework/core/api/router"
	"net/http"
	"os"
	"sync"
	"time"
)

type VDirectory struct {
	fs   http.FileSystem
	name string
}

type VFile struct {
	Compressed string
	FileSize   int64
	ModifyTime int64
	IsFolder   bool

	Data     []byte
	FileName string
}

func (dir VDirectory) Open(name string) (http.File, error) {
	return dir.fs.Open(dir.name + name)
}

func (f *VFile) File() (http.File, error) {
	type httpFile struct {
		*bytes.Reader
		*VFile
	}
	return &httpFile{
		Reader: bytes.NewReader(f.Data),
		VFile:  f,
	}, nil
}

func (f *VFile) Close() error {
	return nil
}

func (f *VFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f *VFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *VFile) Name() string {
	return f.FileName
}

func (f *VFile) Size() int64 {
	return f.FileSize
}

func (f *VFile) Mode() os.FileMode {
	return 0
}

func (f *VFile) ModTime() time.Time {
	return time.Unix(f.ModifyTime, 0)
}

func (f *VFile) IsDir() bool {
	return f.IsFolder
}

func (f *VFile) Sys() interface{} {
	return f
}
func VFS() http.FileSystem {
	return VirtualFS{}
}

type VirtualFS struct{}

var vfs []http.FileSystem
var lock sync.Mutex

func RegisterFS(fs http.FileSystem) {
	lock.Lock()
	vfs = append([]http.FileSystem{fs}, vfs...)
	lock.Unlock()
}

func (VirtualFS) Open(name string) (http.File, error) {

	for _, v := range vfs {
		f1, err := v.Open(name)
		if err == nil {
			return f1, err
		}
	}
	return nil, os.ErrNotExist
}

func main() {
	os.Mkdir("web", 0777)
	RegisterFS(StaticFS{StaticFolder: "web", TrimLeftPath: "/web", CheckLocalFirst: true})

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(VFS()))
	router := httprouter.New(mux)
	err := http.ListenAndServe(":8989", context.ClearHandler(router))
	if err != nil {
		panic(err)
	}
}
