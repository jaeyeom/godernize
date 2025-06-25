package a

import (
	"fmt"
	"os"
)

func testIsNotExist() {
	file, err := os.Open("nonexistent.txt")
	if err != nil {
		if os.IsNotExist(err) { // want "Replace multiple deprecated os error functions with modern errors.Is\\(\\) patterns"
			fmt.Println("File does not exist")
		}
	}
	_ = file
}

func testIsExist() {
	err := os.Mkdir("test", 0755)
	if err != nil {
		if os.IsExist(err) {
			fmt.Println("Directory already exists")
		}
	}
}

func testIsPermission() {
	file, err := os.Open("/root/secret.txt")
	if err != nil {
		if os.IsPermission(err) {
			fmt.Println("Permission denied")
		}
	}
	_ = file
}

//godernize:ignore
func ignoreAll() {
	_, err := os.Stat("ignored.txt")
	if os.IsNotExist(err) { // This should be ignored
		fmt.Println("Ignored check")
	}
}

//godernize:ignore=oserrors
func ignoreOsErrors() {
	_, err := os.Stat("specific.txt")
	if os.IsNotExist(err) { // This should be ignored
		fmt.Println("Specific check ignored")
	}
}

//godernize:ignore=IsNotExist
func ignoreSpecificFunction() {
	_, err := os.Stat("specific.txt")
	if os.IsNotExist(err) { // This should be ignored
		fmt.Println("Function-specific check ignored")
	}
	if os.IsExist(err) { // want "os.IsExist is deprecated, use errors.Is\\(err, fs.ErrExist\\) instead"
		fmt.Println("This should still be reported")
	}
}

func ignoreWithComment() {
	_, err := os.Stat("comment.txt")
	//godernize:ignore
	if os.IsNotExist(err) { // This should be ignored
		fmt.Println("Comment ignored")
	}
}
