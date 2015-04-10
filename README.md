esc
===

A file embedder for Go.

Godoc: http://godoc.org/github.com/mjibson/esc

## Examples

### Serving embedded files with HTTP

Assuming you have a directory called `static` and have run `esc static > static.go` and main.go has the following
code:

```Go
package main

import (
    "log"
    "net/http"
)

func main() {

    //FS() is created by esc and returns a http.Filesystem compatible with http.FileServer
    http.Handle("/static/", http.FileServer(FS(false)))

    //Start the server
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatal("HTTP Server failed: ", err)
    }
}
```

You can now execute `go run main.go static.go` and access the embedded files in a browser via `http://localhost:8080/static/`
