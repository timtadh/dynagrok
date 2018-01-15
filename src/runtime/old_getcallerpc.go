// +build !go1.9

package runtime

import "unsafe"

// GetCallerPC, finds the PC (program counter) of the function
// that calls this function. So if you have
//
// 	func foo() int {
// 		bar(7)
// 		return 1
// 	}
//
// 	func bar(wacky int) {
// 		runtime.GetCallerPC(unsafe.Pointer(&wacky))
// 	}
//
// you will get the pc of `return 1` in foo. This works a lot like
// the built-in Caller() function but is massively less safe calling
// the compiler intrinsic getcallerpc(.) directly.
func GetCallerPC(arg0 unsafe.Pointer) uintptr {
	return uintptr(getcallerpc(arg0))
}
