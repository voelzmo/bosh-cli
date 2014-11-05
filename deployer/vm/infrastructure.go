package vm

import (
	bmstemcell "github.com/cloudfoundry/bosh-micro-cli/deployer/stemcell"
)

type Infrastructure interface {
	CreateVM(bmstemcell.CID, map[string]interface{}, map[string]interface{}, map[string]interface{}) (CID, error)
}
