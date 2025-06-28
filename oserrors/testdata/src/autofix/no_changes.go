package autofix

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

var _ = noChanges

func noChanges() {
	_, err := os.Open("test.txt")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Println("File does not exist - already modern")
		}
	}
}
