package utilities

// globally accessible utilities

import (
	"log"
)


// Check Error and log
func CE(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Template struct
type Template_vars struct {
	Pgchannel string
	Table_name string
	Json_column string
}

