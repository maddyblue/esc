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

type _esc_file struct {
	data  []byte
	local string
}

func main() {
	flag.Parse()
	var err error
	var fnames, dirnames []string
	content := make(map[string]_esc_file)
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
				content[n] = _esc_file{data: b, local: fpath}
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
	dirs := map[string]bool{"/": true}
	for _, fname := range fnames {
		f := content[fname]
		for b := path.Dir(fname); b != "/"; b = path.Dir(b) {
			dirs[b] = true
		}
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
		local: %q,
		size:  %v,
		compressed: %s,
	},%s`, fname, f.local, len(f.data), segment(&buf), "\n")
	}
	for d := range dirs {
		dirnames = append(dirnames, d)
	}
	sort.Strings(dirnames)
	for _, dir := range dirnames {
		fmt.Fprintf(w, `
	%q: {
		isDir: true,
	},%s`, dir, "\n")
	}
	fmt.Fprint(w, footer)
}

func segment(s *bytes.Buffer) string {
	b := new(bytes.Buffer)
	first := true
	for s.Len() > 0 {
		v := string(s.Next(100))
		if !first {
			b.WriteString("\t\t\t")
		} else {
			first = false
		}
		b.WriteString(fmt.Sprintf("%q", v))
		if s.Len() > 0 {
			b.WriteString(" +\n")
		}
	}
	return b.String()
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

type _esc_localFS struct{}

var _esc_local _esc_localFS

type _esc_staticFS struct{}

var _esc_static _esc_staticFS

type _esc_file struct {
	compressed string
	size       int64
	local      string
	isDir      bool

	data []byte
	once sync.Once
	name string
}

func (_esc_localFS) Open(name string) (http.File, error) {
	f, present := _esc_data[name]
	if !present {
		return nil, os.ErrNotExist
	}
	return os.Open(f.local)
}

func (_esc_staticFS) Open(name string) (http.File, error) {
	f, present := _esc_data[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	var err error
	f.once.Do(func() {
		f.name = path.Base(name)
		if f.size == 0 {
			return
		}
		var gr *gzip.Reader
		gr, err = gzip.NewReader(bytes.NewBufferString(f.compressed))
		if err != nil {
			return
		}
		f.data, err = ioutil.ReadAll(gr)
	})
	if err != nil {
		return nil, err
	}
	return f.File()
}

func (f *_esc_file) File() (http.File, error) {
	type httpFile struct {
		*bytes.Reader
		*_esc_file
	}
	return &httpFile{
		Reader:    bytes.NewReader(f.data),
		_esc_file: f,
	}, nil
}

func (f *_esc_file) Close() error {
	return nil
}

func (f *_esc_file) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f *_esc_file) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *_esc_file) Name() string {
	return f.name
}

func (f *_esc_file) Size() int64 {
	return f.size
}

func (f *_esc_file) Mode() os.FileMode {
	return 0
}

func (f *_esc_file) ModTime() time.Time {
	return time.Time{}
}

func (f *_esc_file) IsDir() bool {
	return f.isDir
}

func (f *_esc_file) Sys() interface{} {
	return f
}

// FS returns a http.Filesystem for the embedded assets. If useLocal is true,
// the filesystem's contents are instead used.
func FS(useLocal bool) http.FileSystem {
	if useLocal {
		return _esc_local
	}
	return _esc_static
}

var _esc_data = map[string]*_esc_file{
`
	footer = `}
`
)
