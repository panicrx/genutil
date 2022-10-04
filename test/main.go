package main

import (
	"github.com/panicrx/genutil"
)

func main() {
	pkg, file, tn, err := genutil.LoadPackageAndFindClosestType()
	if err != nil {
		panic(err)
	}

	_, _, _ = pkg, file, tn
}
