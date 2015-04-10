/*

esc embeds files into go programs and provides http.FileSystem interfaces
to them.

It adds all named files or files recursively under named directories at the
path specified. The output file provides an http.FileSystem interface with
zero dependencies on packages outside the standard library.

Usage:
    esc [-o outfile.go] [-pkg package] [-prefix prefix] [name ...]

The flags are:
    -o=""
        output filename, defaults to stdout
    -pkg="main"
        package name of output file, defaults to main
    -prefix=""
        strip given prefix from filenames

Accessing Embedded Files

After producing an output file, the assets may be accessed with the FS()
function, which takes a flag to use local assets instead (for local
development).

HTTP Example

Embedded assets can be served with HTTP using the http.FileServer. Assuming you have a directory called "static" and
have run "esc static > static.go" and main.go has the following code:

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


You can now execute "go run main.go static.go" and access the embedded files in a
browser via http://localhost:8080/static/

*/
package main
