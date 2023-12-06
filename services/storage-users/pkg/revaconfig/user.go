package revaconfig

import "github.com/owncloud/ocis/v2/services/storage-users/pkg/config"

// StorageProviderDrivers are the drivers for the storage provider
func StorageProviderDrivers(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"eos":         EOS(cfg),
		"eoshome":     EOSHome(cfg),
		"eosgrpc":     EOSGRPC(cfg),
		"local":       Local(cfg),
		"localhome":   LocalHome(cfg),
		"owncloudsql": OwnCloudSQL(cfg),
		"ocis":        OcisNoEvents(cfg),
		"s3":          S3(cfg),
		"s3ng":        S3NGNoEvents(cfg),
		"posix":       Posix(cfg),
	}
}

// DataProviderDrivers are the drivers for the storage provider
func DataProviderDrivers(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"eos":         EOS(cfg),
		"eoshome":     EOSHome(cfg),
		"eosgrpc":     EOSGRPC(cfg),
		"local":       Local(cfg),
		"localhome":   LocalHome(cfg),
		"owncloudsql": OwnCloudSQL(cfg),
		"ocis":        Ocis(cfg),
		"s3":          S3(cfg),
		"s3ng":        S3NG(cfg),
		"posix":       Posix(cfg),
	}
}
