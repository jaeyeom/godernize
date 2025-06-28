package autofix

import (
	"fmt"
	"os"
)

var _ = keepOSImport

func keepOSImport() {
	file, err := os.Open("test.txt")
	if err != nil {
		if os.IsNotExist(err) { // want `os.IsNotExist is deprecated, use errors.Is\(err, fs.ErrNotExist\) instead`
			fmt.Println("File does not exist")
		}
	}
	defer file.Close()

	// os package is still used for other purposes
	info, _ := os.Stat("test.txt")
	fmt.Println(info.Size())
}
