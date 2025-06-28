package autofix

import (
	"fmt"
	"os"
)

var _ = multipleDeprecated

func multipleDeprecated() {
	var err error
	if os.IsNotExist(err) { // want `os.IsNotExist is deprecated, use errors.Is\(err, fs.ErrNotExist\) instead`
		fmt.Println("File does not exist")
	}
	if os.IsExist(err) { // want `os.IsExist is deprecated, use errors.Is\(err, fs.ErrExist\) instead`
		fmt.Println("File exists")
	}
	if os.IsPermission(err) { // want `os.IsPermission is deprecated, use errors.Is\(err, fs.ErrPermission\) instead`
		fmt.Println("Permission denied")
	}
}
