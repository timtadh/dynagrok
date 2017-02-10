package excludes

import (
	"strings"
)

var excludedPackages = map[string]bool{
	"fmt":             true,
	"runtime":         true,
	"strings":         true,
	"sync":            true,
	"strconv":         true,
	"io":              true,
	"os":              true,
	"unsafe":          true,
	"errors":          true,
	"internal/race":   true,
	"math":            true,
	"syscall":         true,
	"time":            true,
	"reflect":         true,
	"unicode":         true,
	"sort":            true,
	"encoding/json":   true,
	"bytes":           true,
	"path/filepath":   true,
	"encoding/base64": true,
}

func ExcludedPkg(pkg string) bool {
	parts := strings.Split(pkg, "/")
	if len(parts) > 0 && excludedPackages[parts[0]] {
		return true
	}
	return excludedPackages[pkg]
}
