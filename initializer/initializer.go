package initializer

import (
	rice "github.com/GeertJohan/go.rice"
	"github.com/cloudspannerecosystem/dynamodb-adapter/config"
	"github.com/cloudspannerecosystem/dynamodb-adapter/service/services"
	"github.com/cloudspannerecosystem/dynamodb-adapter/service/spanner"
	"github.com/cloudspannerecosystem/dynamodb-adapter/storage"
)

// InitAll - this will initiliaze all the project object
// Config, storage and all other global objects are initiliaze
func InitAll(box *rice.Box) error {
	config.InitConfig(box)
	storage.InitliazeDriver()
	err := spanner.ParseDDL(true)
	if err != nil {
		return err
	}
	services.StartConfigManager()
	services.InitStream()
	return nil
}
