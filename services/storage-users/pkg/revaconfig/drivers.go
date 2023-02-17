package revaconfig

import "github.com/owncloud/ocis/v2/services/storage-users/pkg/config"

// EOS is the config mapping for the EOS storage driver
func EOS(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"namespace":              cfg.Drivers.EOS.Root,
		"shadow_namespace":       cfg.Drivers.EOS.ShadowNamespace,
		"uploads_namespace":      cfg.Drivers.EOS.UploadsNamespace,
		"share_folder":           cfg.Drivers.EOS.ShareFolder,
		"eos_binary":             cfg.Drivers.EOS.EosBinary,
		"xrdcopy_binary":         cfg.Drivers.EOS.XrdcopyBinary,
		"master_url":             cfg.Drivers.EOS.MasterURL,
		"slave_url":              cfg.Drivers.EOS.SlaveURL,
		"cache_directory":        cfg.Drivers.EOS.CacheDirectory,
		"sec_protocol":           cfg.Drivers.EOS.SecProtocol,
		"keytab":                 cfg.Drivers.EOS.Keytab,
		"single_username":        cfg.Drivers.EOS.SingleUsername,
		"user_layout":            cfg.Drivers.EOS.UserLayout,
		"enable_logging":         cfg.Drivers.EOS.EnableLogging,
		"show_hidden_sys_files":  cfg.Drivers.EOS.ShowHiddenSysFiles,
		"force_single_user_mode": cfg.Drivers.EOS.ForceSingleUserMode,
		"use_keytab":             cfg.Drivers.EOS.UseKeytab,
		"gatewaysvc":             cfg.Drivers.EOS.GatewaySVC,
	}
}

// EOSHome is the config mapping for the EOSHome storage driver
func EOSHome(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"namespace":              cfg.Drivers.EOS.Root,
		"shadow_namespace":       cfg.Drivers.EOS.ShadowNamespace,
		"uploads_namespace":      cfg.Drivers.EOS.UploadsNamespace,
		"share_folder":           cfg.Drivers.EOS.ShareFolder,
		"eos_binary":             cfg.Drivers.EOS.EosBinary,
		"xrdcopy_binary":         cfg.Drivers.EOS.XrdcopyBinary,
		"master_url":             cfg.Drivers.EOS.MasterURL,
		"slave_url":              cfg.Drivers.EOS.SlaveURL,
		"cache_directory":        cfg.Drivers.EOS.CacheDirectory,
		"sec_protocol":           cfg.Drivers.EOS.SecProtocol,
		"keytab":                 cfg.Drivers.EOS.Keytab,
		"single_username":        cfg.Drivers.EOS.SingleUsername,
		"user_layout":            cfg.Drivers.EOS.UserLayout,
		"enable_logging":         cfg.Drivers.EOS.EnableLogging,
		"show_hidden_sys_files":  cfg.Drivers.EOS.ShowHiddenSysFiles,
		"force_single_user_mode": cfg.Drivers.EOS.ForceSingleUserMode,
		"use_keytab":             cfg.Drivers.EOS.UseKeytab,
		"gatewaysvc":             cfg.Drivers.EOS.GatewaySVC,
		"spaces_config":          cfg.Drivers.EOS.SpacesConfig,
	}
}

// EOSGRPC is the config mapping for the EOSGRPC storage driver
func EOSGRPC(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"namespace":              cfg.Drivers.EOS.Root,
		"shadow_namespace":       cfg.Drivers.EOS.ShadowNamespace,
		"share_folder":           cfg.Drivers.EOS.ShareFolder,
		"eos_binary":             cfg.Drivers.EOS.EosBinary,
		"xrdcopy_binary":         cfg.Drivers.EOS.XrdcopyBinary,
		"master_url":             cfg.Drivers.EOS.MasterURL,
		"master_grpc_uri":        cfg.Drivers.EOS.GRPCURI,
		"slave_url":              cfg.Drivers.EOS.SlaveURL,
		"cache_directory":        cfg.Drivers.EOS.CacheDirectory,
		"sec_protocol":           cfg.Drivers.EOS.SecProtocol,
		"keytab":                 cfg.Drivers.EOS.Keytab,
		"single_username":        cfg.Drivers.EOS.SingleUsername,
		"user_layout":            cfg.Drivers.EOS.UserLayout,
		"enable_logging":         cfg.Drivers.EOS.EnableLogging,
		"show_hidden_sys_files":  cfg.Drivers.EOS.ShowHiddenSysFiles,
		"force_single_user_mode": cfg.Drivers.EOS.ForceSingleUserMode,
		"use_keytab":             cfg.Drivers.EOS.UseKeytab,
		"enable_home":            false,
		"gatewaysvc":             cfg.Drivers.EOS.GatewaySVC,
		"spaces_config":          cfg.Drivers.EOS.SpacesConfig,
	}
}

// Local is the config mapping for the Local storage driver
func Local(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"root":         cfg.Drivers.Local.Root,
		"share_folder": cfg.Drivers.Local.ShareFolder,
	}
}

// LocalHome is the config mapping for the LocalHome storage driver
func LocalHome(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"root":         cfg.Drivers.Local.Root,
		"share_folder": cfg.Drivers.Local.ShareFolder,
		"user_layout":  cfg.Drivers.Local.UserLayout,
	}
}

// OwnCloudSQL is the config mapping for the OwnCloudSQL storage driver
func OwnCloudSQL(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"datadirectory":   cfg.Drivers.OwnCloudSQL.Root,
		"upload_info_dir": cfg.Drivers.OwnCloudSQL.UploadInfoDir,
		"share_folder":    cfg.Drivers.OwnCloudSQL.ShareFolder,
		"user_layout":     cfg.Drivers.OwnCloudSQL.UserLayout,
		"enable_home":     false,
		"dbusername":      cfg.Drivers.OwnCloudSQL.DBUsername,
		"dbpassword":      cfg.Drivers.OwnCloudSQL.DBPassword,
		"dbhost":          cfg.Drivers.OwnCloudSQL.DBHost,
		"dbport":          cfg.Drivers.OwnCloudSQL.DBPort,
		"dbname":          cfg.Drivers.OwnCloudSQL.DBName,
		"userprovidersvc": cfg.Drivers.OwnCloudSQL.UsersProviderEndpoint,
	}
}

// Ocis is the config mapping for the Ocis storage driver
func Ocis(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"root":                        cfg.Drivers.OCIS.Root,
		"user_layout":                 cfg.Drivers.OCIS.UserLayout,
		"share_folder":                cfg.Drivers.OCIS.ShareFolder,
		"personalspacealias_template": cfg.Drivers.OCIS.PersonalSpaceAliasTemplate,
		"generalspacealias_template":  cfg.Drivers.OCIS.GeneralSpaceAliasTemplate,
		"treetime_accounting":         true,
		"treesize_accounting":         true,
		"permissionssvc":              cfg.Drivers.OCIS.PermissionsEndpoint,
		"permissionssvc_tls_mode":     cfg.Commons.GRPCClientTLS.Mode,
		"max_acquire_lock_cycles":     cfg.Drivers.OCIS.MaxAcquireLockCycles,
		"lock_cycle_duration_factor":  cfg.Drivers.OCIS.LockCycleDurationFactor,
		"asyncfileuploads":            cfg.Drivers.OCIS.AsyncUploads,
		"statcache": map[string]interface{}{
			"cache_store":    cfg.Cache.Store,
			"cache_nodes":    cfg.Cache.Nodes,
			"cache_database": cfg.Cache.Database,
		},
		"events": map[string]interface{}{
			"natsaddress":          cfg.Events.Addr,
			"natsclusterid":        cfg.Events.ClusterID,
			"tlsinsecure":          cfg.Events.TLSInsecure,
			"tlsrootcacertificate": cfg.Events.TLSRootCaCertPath,
			"numconsumers":         cfg.Events.NumConsumers,
		},
		"tokens": map[string]interface{}{
			"transfer_shared_secret": cfg.Commons.TransferSecret,
			"transfer_expires":       cfg.TransferExpires,
			"download_endpoint":      cfg.DataServerURL,
			"datagateway_endpoint":   cfg.DataGatewayURL,
		},
	}
}

// OcisNoEvents is the config mapping for the ocis storage driver emitting no events
func OcisNoEvents(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"root":                        cfg.Drivers.OCIS.Root,
		"user_layout":                 cfg.Drivers.OCIS.UserLayout,
		"share_folder":                cfg.Drivers.OCIS.ShareFolder,
		"personalspacealias_template": cfg.Drivers.OCIS.PersonalSpaceAliasTemplate,
		"generalspacealias_template":  cfg.Drivers.OCIS.GeneralSpaceAliasTemplate,
		"treetime_accounting":         true,
		"treesize_accounting":         true,
		"permissionssvc":              cfg.Drivers.OCIS.PermissionsEndpoint,
		"permissionssvc_tls_mode":     cfg.Commons.GRPCClientTLS.Mode,
		"max_acquire_lock_cycles":     cfg.Drivers.OCIS.MaxAcquireLockCycles,
		"lock_cycle_duration_factor":  cfg.Drivers.OCIS.LockCycleDurationFactor,
		"statcache": map[string]interface{}{
			"cache_store":    cfg.Cache.Store,
			"cache_nodes":    cfg.Cache.Nodes,
			"cache_database": cfg.Cache.Database,
		},
	}
}

// S3 is the config mapping for the s3 storage driver
func S3(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"enable_home": false,
		"region":      cfg.Drivers.S3.Region,
		"access_key":  cfg.Drivers.S3.AccessKey,
		"secret_key":  cfg.Drivers.S3.SecretKey,
		"endpoint":    cfg.Drivers.S3.Endpoint,
		"bucket":      cfg.Drivers.S3.Bucket,
		"prefix":      cfg.Drivers.S3.Root,
	}
}

// S3NG is the config mapping for the s3ng storage driver
func S3NG(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"root":                        cfg.Drivers.S3NG.Root,
		"user_layout":                 cfg.Drivers.S3NG.UserLayout,
		"share_folder":                cfg.Drivers.S3NG.ShareFolder,
		"personalspacealias_template": cfg.Drivers.S3NG.PersonalSpaceAliasTemplate,
		"generalspacealias_template":  cfg.Drivers.S3NG.GeneralSpaceAliasTemplate,
		"treetime_accounting":         true,
		"treesize_accounting":         true,
		"permissionssvc":              cfg.Drivers.S3NG.PermissionsEndpoint,
		"permissionssvc_tls_mode":     cfg.Commons.GRPCClientTLS.Mode,
		"s3.region":                   cfg.Drivers.S3NG.Region,
		"s3.access_key":               cfg.Drivers.S3NG.AccessKey,
		"s3.secret_key":               cfg.Drivers.S3NG.SecretKey,
		"s3.endpoint":                 cfg.Drivers.S3NG.Endpoint,
		"s3.bucket":                   cfg.Drivers.S3NG.Bucket,
		"max_acquire_lock_cycles":     cfg.Drivers.S3NG.MaxAcquireLockCycles,
		"lock_cycle_duration_factor":  cfg.Drivers.S3NG.LockCycleDurationFactor,
		"asyncfileuploads":            cfg.Drivers.OCIS.AsyncUploads,
		"statcache": map[string]interface{}{
			"cache_store":    cfg.Cache.Store,
			"cache_nodes":    cfg.Cache.Nodes,
			"cache_database": cfg.Cache.Database,
		},
		"events": map[string]interface{}{
			"natsaddress":          cfg.Events.Addr,
			"natsclusterid":        cfg.Events.ClusterID,
			"tlsinsecure":          cfg.Events.TLSInsecure,
			"tlsrootcacertificate": cfg.Events.TLSRootCaCertPath,
			"numconsumers":         cfg.Events.NumConsumers,
		},
		"tokens": map[string]interface{}{
			"transfer_shared_secret": cfg.Commons.TransferSecret,
			"transfer_expires":       cfg.TransferExpires,
			"download_endpoint":      cfg.DataServerURL,
			"datagateway_endpoint":   cfg.DataGatewayURL,
		},
	}
}

// S3NGNoEvents is the config mapping for the s3ng storage driver emitting no events
func S3NGNoEvents(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"root":                        cfg.Drivers.S3NG.Root,
		"user_layout":                 cfg.Drivers.S3NG.UserLayout,
		"share_folder":                cfg.Drivers.S3NG.ShareFolder,
		"personalspacealias_template": cfg.Drivers.S3NG.PersonalSpaceAliasTemplate,
		"generalspacealias_template":  cfg.Drivers.S3NG.GeneralSpaceAliasTemplate,
		"treetime_accounting":         true,
		"treesize_accounting":         true,
		"permissionssvc":              cfg.Drivers.S3NG.PermissionsEndpoint,
		"permissionssvc_tls_mode":     cfg.Commons.GRPCClientTLS.Mode,
		"s3.region":                   cfg.Drivers.S3NG.Region,
		"s3.access_key":               cfg.Drivers.S3NG.AccessKey,
		"s3.secret_key":               cfg.Drivers.S3NG.SecretKey,
		"s3.endpoint":                 cfg.Drivers.S3NG.Endpoint,
		"s3.bucket":                   cfg.Drivers.S3NG.Bucket,
		"max_acquire_lock_cycles":     cfg.Drivers.S3NG.MaxAcquireLockCycles,
		"lock_cycle_duration_factor":  cfg.Drivers.S3NG.LockCycleDurationFactor,
		"statcache": map[string]interface{}{
			"cache_store":    cfg.Cache.Store,
			"cache_nodes":    cfg.Cache.Nodes,
			"cache_database": cfg.Cache.Database,
		},
	}
}
