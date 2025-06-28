package autofix

import (
	"fmt"
	"os"
	"strings"
)

var _ = importGrouping

func importGrouping() {
	data := strings.TrimSpace("  hello  ")
	fmt.Println(data)

	var err error
	if os.IsNotExist(err) { // want `os.IsNotExist is deprecated, use errors.Is\(err, fs.ErrNotExist\) instead`
		fmt.Println("File does not exist")
	}
}
