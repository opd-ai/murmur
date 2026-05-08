//go:build android || ios

package main

// Blank-import golang.org/x/mobile/app so gomobile treats this package as a
// mobile application entrypoint when targeting Android or iOS.
import _ "golang.org/x/mobile/app"
