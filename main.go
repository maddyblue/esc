package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

var (
	flagOut    = flag.String("o", "", "Output file, else stdout.")
	flagPkg    = flag.String("pkg", "main", "Package.")
	flagPrefix = flag.String("prefix", "", "Prefix to strip from filesnames.")
)

type file struct {
	data  []byte
	local string
}

func main() {
	flag.Parse()
	var err error
	var fnames []string
	content := make(map[string]file)
	prefix := filepath.ToSlash(*flagPrefix)
	for _, base := range flag.Args() {
		files := []string{base}
		for len(files) > 0 {
			fname := files[0]
			files = files[1:]
			f, err := os.Open(fname)
			if err != nil {
				log.Fatal(err)
			}
			fi, err := f.Stat()
			if err != nil {
				log.Fatal(err)
			}
			if fi.IsDir() {
				fis, err := f.Readdir(0)
				if err != nil {
					log.Fatal(err)
				}
				for _, fi := range fis {
					files = append(files, filepath.Join(fname, fi.Name()))
				}
			} else {
				b, err := ioutil.ReadAll(f)
				if err != nil {
					log.Fatal(err)
				}
				fpath := filepath.ToSlash(fname)
				n := strings.TrimPrefix(fpath, prefix)
				n = path.Join("/", n)
				content[n] = file{data: b, local: fpath}
				fnames = append(fnames, n)
			}
			f.Close()
		}
	}
	sort.Strings(fnames)
	w := os.Stdout
	if *flagOut != "" {
		if w, err = os.Create(*flagOut); err != nil {
			log.Fatal(err)
		}
		defer w.Close()
	}
	fmt.Fprintf(w, header, *flagPkg)
	for _, fname := range fnames {
		f := content[fname]
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		if _, err := gw.Write(f.data); err != nil {
			log.Fatal(err)
		}
		if err := gw.Close(); err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, `
	%q: {
		local:      %q,
		size:       %v,
		compressed: %q,
	},%s`, fname, f.local, len(f.data), buf.String(), "\n")
	}
	}
	fmt.Fprint(w, footer)
}

const (
	header = `package %s

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

type localFS struct{}

var local localFS

type staticFS struct{}

var static staticFS

type file struct {
	compressed string
	size       int64
	local      string

	data []byte
	once sync.Once
	name string
}

func (_ localFS) Open(name string) (http.File, error) {
	f, present := data[name]
	if !present {
		return nil, os.ErrNotExist
	}
	return os.Open(f.local)
}

func (_ staticFS) Open(name string) (http.File, error) {
	f, present := data[name]
	if !present {
		return nil, os.ErrNotExist
	}
	var err error
	f.once.Do(func() {
		var gr *gzip.Reader
		gr, err = gzip.NewReader(bytes.NewBufferString(f.compressed))
		if err != nil {
			return
		}
		f.data, err = ioutil.ReadAll(gr)
		f.name = path.Base(name)
	})
	if err != nil {
		return nil, err
	}
	return f.File()
}

func (f *file) File() (http.File, error) {
	return &httpFile{
		Reader: bytes.NewReader(f.data),
		file:   f,
	}, nil
}

type httpFile struct {
	*bytes.Reader
	*file
}

func (f *file) Close() error {
	return nil
}

func (f *file) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f *file) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *file) Name() string {
	return f.name
}

func (f *file) Size() int64 {
	return f.size
}

func (f *file) Mode() os.FileMode {
	return 0
}

func (f *file) ModTime() time.Time {
	return time.Time{}
}
func (f *file) IsDir() bool {
	return false
}
func (f *file) Sys() interface{} {
	return f
}

// FS returns a http.Filesystem for the embedded assets. If useLocal is true,
// the filesystem's contents are instead used.
func FS(useLocal bool) http.FileSystem {
	if useLocal {
		return local
	}
	return static
}

var data = map[string]*file{
`
	footer = `}
`
)
