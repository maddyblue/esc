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

FS(Must)?(Byte|String) return an asset as a (byte slice|string).
FSMust(Byte|String) panics if the asset is not found.

*/
package main
