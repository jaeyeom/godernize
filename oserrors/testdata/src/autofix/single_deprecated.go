package autofix

import (
	"fmt"
	"os"
)

var _ = singleDeprecated

func singleDeprecated() {
	var err error
	if os.IsNotExist(err) { // want `os.IsNotExist is deprecated, use errors.Is\(err, fs.ErrNotExist\) instead`
		fmt.Println("File does not exist")
	}
}
