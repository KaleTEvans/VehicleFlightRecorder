//go:build tools

package mobile

// Retain the matching binding runtime in the module graph for gomobile bind.
import _ "golang.org/x/mobile/bind"
