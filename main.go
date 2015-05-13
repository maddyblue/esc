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

type escFile struct {
	data  []byte
	local string
}

func main() {
	flag.Parse()
	var err error
	var fnames, dirnames []string
	content := make(map[string]escFile)
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
				content[n] = escFile{data: b, local: fpath}
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
		local := path.Join(prefix, dir)
		if len(local) == 0 {
			local = "."
		}
		fmt.Fprintf(w, `
	%q: {
		isDir: true,
		local: %q,
	},%s`, dir, local, "\n")
	}
	fmt.Fprint(w, footer)
}

func segment(s *bytes.Buffer) string {
	b := bytes.NewBufferString("\"\" +\n")
	for s.Len() > 0 {
		v := string(s.Next(100))
		b.WriteString(fmt.Sprintf("\t\t\t%q", v))
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

type escLocalFS struct{}

var escLocal escLocalFS

type escStaticFS struct{}

var escStatic escStaticFS

type escFile struct {
	compressed string
	size       int64
	local      string
	isDir      bool

	data []byte
	once sync.Once
	name string
}

func (escLocalFS) Open(name string) (http.File, error) {
	f, present := escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	return os.Open(f.local)
}

func (escStaticFS) prepare(name string) (*escFile, error) {
	f, present := escData[path.Clean(name)]
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
	return f, nil
}

func (fs escStaticFS) Open(name string) (http.File, error) {
	f, err := fs.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.File()
}

func (f *escFile) File() (http.File, error) {
	type httpFile struct {
		*bytes.Reader
		*escFile
	}
	return &httpFile{
		Reader:  bytes.NewReader(f.data),
		escFile: f,
	}, nil
}

func (f *escFile) Close() error {
	return nil
}

func (f *escFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f *escFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *escFile) Name() string {
	return f.name
}

func (f *escFile) Size() int64 {
	return f.size
}

func (f *escFile) Mode() os.FileMode {
	return 0
}

func (f *escFile) ModTime() time.Time {
	return time.Time{}
}

func (f *escFile) IsDir() bool {
	return f.isDir
}

func (f *escFile) Sys() interface{} {
	return f
}

// FS returns a http.Filesystem for the embedded assets. If useLocal is true,
// the filesystem's contents are instead used.
func FS(useLocal bool) http.FileSystem {
	if useLocal {
		return escLocal
	}
	return escStatic
}

// FSByte returns the named file from the embedded assets. If useLocal is
// true, the filesystem's contents are instead used.
func FSByte(useLocal bool, name string) ([]byte, error) {
	if useLocal {
		f, err := escLocal.Open(name)
		if err != nil {
			return nil, err
		}
		return ioutil.ReadAll(f)
	}
	f, err := escStatic.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.data, nil
}

// FSMustByte is the same as FSByte, but panics if name is not present.
func FSMustByte(useLocal bool, name string) []byte {
	b, err := FSByte(useLocal, name)
	if err != nil {
		panic(err)
	}
	return b
}

// FSString is the string version of FSByte.
func FSString(useLocal bool, name string) (string, error) {
	b, err := FSByte(useLocal, name)
	return string(b), err
}

// FSMustString is the string version of FSMustByte.
func FSMustString(useLocal bool, name string) string {
	return string(FSMustByte(useLocal, name))
}

var escData = map[string]*escFile{
`
	footer = `}
`
)
