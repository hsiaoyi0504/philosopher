package cdhit

import (
	"errors"
	"os"

	"github.com/Nesvilab/philosopher/lib/msg"

	"github.com/Nesvilab/philosopher/lib/sys"
)

// Unix64 deploys CD-HIT
func Unix64(unix64 string) {

	bin, e1 := Asset("cd-hit")
	if e1 != nil {
		msg.DeployAsset(errors.New("CD-HIT"), "Cannot read CD-HIT obo")
	}

	e2 := os.WriteFile(unix64, bin, sys.FilePermission())
	if e2 != nil {
		msg.DeployAsset(errors.New("CD-HIT"), "Cannot deploy CD-HIT 64-bit")
	}
}
