package main

import (
	"fmt"
	"github.com/golibry/go-fs"
	"os"
	"path"
)

func main() {
	test := fs.New(path.Join(os.TempDir(), "go-fs-test.lock"))
	fmt.Println(test)

	err := test.Lock()
	fmt.Println(err) // should be nil

	err = test.Lock()
	fmt.Println(err)             // should be ErrLockHeld
	fmt.Println(test.IsLocked()) // should be true

	_ = test.Unlock()
}
