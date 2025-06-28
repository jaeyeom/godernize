// Package directive provides functionality to parse ignore directives in comments.
package directive

import (
	"go/ast"
	"slices"
	"strings"
)

// Ignore represent a special instruction embedded in the source code.
//
// The directive can be as simple as
//
//	//godernize:ignore
//
// or consist of name
//
//	//godernize:ignore=oserrors
//	//godernize:ignore=IsNotExist
//
// or multiple names
//
//	//godernize:ignore=IsNotExist,IsExist
type Ignore struct {
	Names []string
}

// ParseIgnore parse the directive from the comments.
func ParseIgnore(doc *ast.CommentGroup) *Ignore {
	if doc == nil {
		return nil
	}

	for _, comment := range doc.List {
		text := strings.TrimSpace(comment.Text)
		if text == "//godernize:ignore" {
			return &Ignore{}
		}

		// parse the Names if exists
		if val, found := strings.CutPrefix(text, "//godernize:ignore="); found {
			val = strings.TrimSpace(val)
			if val == "" {
				return &Ignore{}
			}

			names := strings.Split(val, ",")
			if len(names) == 0 {
				continue
			}

			for i, name := range names {
				names[i] = strings.TrimSpace(name)
			}

			if len(names) > 0 {
				return &Ignore{Names: names}
			}

			return &Ignore{}
		}
	}

	return nil
}

func (i *Ignore) hasName(name string) bool {
	return slices.Contains(i.Names, name)
}

// ShouldIgnore return true if the name should be ignored.
func (i *Ignore) ShouldIgnore(name string) bool {
	if len(i.Names) == 0 {
		return true
	}

	return i.hasName(name)
}

// HasSpecificRules returns true if this ignore directive has specific rules
// (e.g., //godernize:ignore=IsNotExist) rather than a general ignore (//godernize:ignore).
func (i *Ignore) HasSpecificRules() bool {
	return len(i.Names) > 0
}
