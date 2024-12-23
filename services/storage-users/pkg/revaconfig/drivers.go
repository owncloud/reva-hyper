package revaconfig

import (
	"github.com/owncloud/ocis/v2/services/storage-users/pkg/config"
)

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
	}
}

// Local is the config mapping for the Local storage driver
func Local(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"root":         cfg.Drivers.Local.Root,
		"share_folder": cfg.Drivers.Local.ShareFolder,
	}
}

// Posix is the config mapping for the Posix storage driver
func Posix(cfg *config.Config, enableFSWatch bool) map[string]interface{} {
	return map[string]interface{}{
		"root":                       cfg.Drivers.Posix.Root,
		"personalspacepath_template": cfg.Drivers.Posix.PersonalSpacePathTemplate,
		"generalspacepath_template":  cfg.Drivers.Posix.GeneralSpacePathTemplate,
		"permissionssvc":             cfg.Drivers.Posix.PermissionsEndpoint,
		"permissionssvc_tls_mode":    cfg.Commons.GRPCClientTLS.Mode,
		"treetime_accounting":        true,
		"treesize_accounting":        true,
		"asyncfileuploads":           cfg.Drivers.Posix.AsyncUploads,
		"scan_debounce_delay":        cfg.Drivers.Posix.ScanDebounceDelay,
		"idcache": map[string]interface{}{
			"cache_store":               cfg.IDCache.Store,
			"cache_nodes":               cfg.IDCache.Nodes,
			"cache_database":            cfg.IDCache.Database,
			"cache_ttl":                 cfg.IDCache.TTL,
			"cache_disable_persistence": cfg.IDCache.DisablePersistence,
			"cache_auth_username":       cfg.IDCache.AuthUsername,
			"cache_auth_password":       cfg.IDCache.AuthPassword,
		},
		"filemetadatacache": map[string]interface{}{
			"cache_store":               cfg.FilemetadataCache.Store,
			"cache_nodes":               cfg.FilemetadataCache.Nodes,
			"cache_database":            cfg.FilemetadataCache.Database,
			"cache_ttl":                 cfg.FilemetadataCache.TTL,
			"cache_disable_persistence": cfg.FilemetadataCache.DisablePersistence,
			"cache_auth_username":       cfg.FilemetadataCache.AuthUsername,
			"cache_auth_password":       cfg.FilemetadataCache.AuthPassword,
		},
		"use_space_groups":           cfg.Drivers.Posix.UseSpaceGroups,
		"watch_fs":                   enableFSWatch,
		"watch_type":                 cfg.Drivers.Posix.WatchType,
		"watch_mount_point":          cfg.Drivers.Posix.WatchMountPoint,
		"watch_path":                 cfg.Drivers.Posix.WatchPath,
		"watch_folder_kafka_brokers": cfg.Drivers.Posix.WatchFolderKafkaBrokers,
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
		"tokens": map[string]interface{}{
			"download_endpoint":      cfg.DataServerURL,
			"datagateway_endpoint":   cfg.DataGatewayURL,
			"transfer_shared_secret": cfg.Commons.TransferSecret,
			"transfer_expires":       cfg.TransferExpires,
		},
	}
}

// Ocis is the config mapping for the Ocis storage driver
func Ocis(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"metadata_backend": "messagepack",
		"propagator":       cfg.Drivers.OCIS.Propagator,
		"async_propagator_options": map[string]interface{}{
			"propagation_delay": cfg.Drivers.OCIS.AsyncPropagatorOptions.PropagationDelay,
		},
		"root":                        cfg.Drivers.OCIS.Root,
		"user_layout":                 cfg.Drivers.OCIS.UserLayout,
		"share_folder":                cfg.Drivers.OCIS.ShareFolder,
		"personalspacealias_template": cfg.Drivers.OCIS.PersonalSpaceAliasTemplate,
		"personalspacepath_template":  cfg.Drivers.OCIS.PersonalSpacePathTemplate,
		"generalspacealias_template":  cfg.Drivers.OCIS.GeneralSpaceAliasTemplate,
		"generalspacepath_template":   cfg.Drivers.OCIS.GeneralSpacePathTemplate,
		"treetime_accounting":         true,
		"treesize_accounting":         true,
		"permissionssvc":              cfg.Drivers.OCIS.PermissionsEndpoint,
		"permissionssvc_tls_mode":     cfg.Commons.GRPCClientTLS.Mode,
		"max_acquire_lock_cycles":     cfg.Drivers.OCIS.MaxAcquireLockCycles,
		"lock_cycle_duration_factor":  cfg.Drivers.OCIS.LockCycleDurationFactor,
		"max_concurrency":             cfg.Drivers.OCIS.MaxConcurrency,
		"asyncfileuploads":            cfg.Drivers.OCIS.AsyncUploads,
		"max_quota":                   cfg.Drivers.OCIS.MaxQuota,
		"disable_versioning":          cfg.Drivers.OCIS.DisableVersioning,
		"filemetadatacache": map[string]interface{}{
			"cache_store":               cfg.FilemetadataCache.Store,
			"cache_nodes":               cfg.FilemetadataCache.Nodes,
			"cache_database":            cfg.FilemetadataCache.Database,
			"cache_ttl":                 cfg.FilemetadataCache.TTL,
			"cache_disable_persistence": cfg.FilemetadataCache.DisablePersistence,
			"cache_auth_username":       cfg.FilemetadataCache.AuthUsername,
			"cache_auth_password":       cfg.FilemetadataCache.AuthPassword,
		},
		"idcache": map[string]interface{}{
			"cache_store":               cfg.IDCache.Store,
			"cache_nodes":               cfg.IDCache.Nodes,
			"cache_database":            cfg.IDCache.Database,
			"cache_ttl":                 cfg.IDCache.TTL,
			"cache_disable_persistence": cfg.IDCache.DisablePersistence,
			"cache_auth_username":       cfg.IDCache.AuthUsername,
			"cache_auth_password":       cfg.IDCache.AuthPassword,
		},
		"events": map[string]interface{}{
			"numconsumers": cfg.Events.NumConsumers,
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
		"metadata_backend": "messagepack",
		"propagator":       cfg.Drivers.OCIS.Propagator,
		"async_propagator_options": map[string]interface{}{
			"propagation_delay": cfg.Drivers.OCIS.AsyncPropagatorOptions.PropagationDelay,
		},
		"root":                        cfg.Drivers.OCIS.Root,
		"user_layout":                 cfg.Drivers.OCIS.UserLayout,
		"share_folder":                cfg.Drivers.OCIS.ShareFolder,
		"personalspacealias_template": cfg.Drivers.OCIS.PersonalSpaceAliasTemplate,
		"personalspacepath_template":  cfg.Drivers.OCIS.PersonalSpacePathTemplate,
		"generalspacealias_template":  cfg.Drivers.OCIS.GeneralSpaceAliasTemplate,
		"generalspacepath_template":   cfg.Drivers.OCIS.GeneralSpacePathTemplate,
		"treetime_accounting":         true,
		"treesize_accounting":         true,
		"permissionssvc":              cfg.Drivers.OCIS.PermissionsEndpoint,
		"permissionssvc_tls_mode":     cfg.Commons.GRPCClientTLS.Mode,
		"max_acquire_lock_cycles":     cfg.Drivers.OCIS.MaxAcquireLockCycles,
		"lock_cycle_duration_factor":  cfg.Drivers.OCIS.LockCycleDurationFactor,
		"max_concurrency":             cfg.Drivers.OCIS.MaxConcurrency,
		"max_quota":                   cfg.Drivers.OCIS.MaxQuota,
		"disable_versioning":          cfg.Drivers.OCIS.DisableVersioning,
		"filemetadatacache": map[string]interface{}{
			"cache_store":               cfg.FilemetadataCache.Store,
			"cache_nodes":               cfg.FilemetadataCache.Nodes,
			"cache_database":            cfg.FilemetadataCache.Database,
			"cache_ttl":                 cfg.FilemetadataCache.TTL,
			"cache_disable_persistence": cfg.FilemetadataCache.DisablePersistence,
			"cache_auth_username":       cfg.FilemetadataCache.AuthUsername,
			"cache_auth_password":       cfg.FilemetadataCache.AuthPassword,
		},
		"idcache": map[string]interface{}{
			"cache_store":               cfg.IDCache.Store,
			"cache_nodes":               cfg.IDCache.Nodes,
			"cache_database":            cfg.IDCache.Database,
			"cache_ttl":                 cfg.IDCache.TTL,
			"cache_disable_persistence": cfg.IDCache.DisablePersistence,
			"cache_auth_username":       cfg.IDCache.AuthUsername,
			"cache_auth_password":       cfg.IDCache.AuthPassword,
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
		"metadata_backend": "messagepack",
		"propagator":       cfg.Drivers.S3NG.Propagator,
		"async_propagator_options": map[string]interface{}{
			"propagation_delay": cfg.Drivers.S3NG.AsyncPropagatorOptions.PropagationDelay,
		},
		"root":                        cfg.Drivers.S3NG.Root,
		"user_layout":                 cfg.Drivers.S3NG.UserLayout,
		"share_folder":                cfg.Drivers.S3NG.ShareFolder,
		"personalspacealias_template": cfg.Drivers.OCIS.PersonalSpaceAliasTemplate,
		"personalspacepath_template":  cfg.Drivers.OCIS.PersonalSpacePathTemplate,
		"generalspacealias_template":  cfg.Drivers.OCIS.GeneralSpaceAliasTemplate,
		"generalspacepath_template":   cfg.Drivers.OCIS.GeneralSpacePathTemplate,
		"treetime_accounting":         true,
		"treesize_accounting":         true,
		"permissionssvc":              cfg.Drivers.S3NG.PermissionsEndpoint,
		"permissionssvc_tls_mode":     cfg.Commons.GRPCClientTLS.Mode,
		"s3.region":                   cfg.Drivers.S3NG.Region,
		"s3.access_key":               cfg.Drivers.S3NG.AccessKey,
		"s3.secret_key":               cfg.Drivers.S3NG.SecretKey,
		"s3.endpoint":                 cfg.Drivers.S3NG.Endpoint,
		"s3.bucket":                   cfg.Drivers.S3NG.Bucket,
		"s3.disable_content_sha254":   cfg.Drivers.S3NG.DisableContentSha256,
		"s3.disable_multipart":        cfg.Drivers.S3NG.DisableMultipart,
		"s3.send_content_md5":         cfg.Drivers.S3NG.SendContentMd5,
		"s3.concurrent_stream_parts":  cfg.Drivers.S3NG.ConcurrentStreamParts,
		"s3.num_threads":              cfg.Drivers.S3NG.NumThreads,
		"s3.part_size":                cfg.Drivers.S3NG.PartSize,
		"max_acquire_lock_cycles":     cfg.Drivers.S3NG.MaxAcquireLockCycles,
		"lock_cycle_duration_factor":  cfg.Drivers.S3NG.LockCycleDurationFactor,
		"max_concurrency":             cfg.Drivers.S3NG.MaxConcurrency,
		"disable_versioning":          cfg.Drivers.S3NG.DisableVersioning,
		"asyncfileuploads":            cfg.Drivers.OCIS.AsyncUploads,
		"filemetadatacache": map[string]interface{}{
			"cache_store":               cfg.FilemetadataCache.Store,
			"cache_nodes":               cfg.FilemetadataCache.Nodes,
			"cache_database":            cfg.FilemetadataCache.Database,
			"cache_ttl":                 cfg.FilemetadataCache.TTL,
			"cache_disable_persistence": cfg.FilemetadataCache.DisablePersistence,
			"cache_auth_username":       cfg.FilemetadataCache.AuthUsername,
			"cache_auth_password":       cfg.FilemetadataCache.AuthPassword,
		},
		"idcache": map[string]interface{}{
			"cache_store":               cfg.IDCache.Store,
			"cache_nodes":               cfg.IDCache.Nodes,
			"cache_database":            cfg.IDCache.Database,
			"cache_ttl":                 cfg.IDCache.TTL,
			"cache_disable_persistence": cfg.IDCache.DisablePersistence,
			"cache_auth_username":       cfg.IDCache.AuthUsername,
			"cache_auth_password":       cfg.IDCache.AuthPassword,
		},
		"events": map[string]interface{}{
			"numconsumers": cfg.Events.NumConsumers,
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
		"metadata_backend": "messagepack",
		"propagator":       cfg.Drivers.S3NG.Propagator,
		"async_propagator_options": map[string]interface{}{
			"propagation_delay": cfg.Drivers.S3NG.AsyncPropagatorOptions.PropagationDelay,
		},
		"root":                        cfg.Drivers.S3NG.Root,
		"user_layout":                 cfg.Drivers.S3NG.UserLayout,
		"share_folder":                cfg.Drivers.S3NG.ShareFolder,
		"personalspacealias_template": cfg.Drivers.OCIS.PersonalSpaceAliasTemplate,
		"personalspacepath_template":  cfg.Drivers.OCIS.PersonalSpacePathTemplate,
		"generalspacealias_template":  cfg.Drivers.OCIS.GeneralSpaceAliasTemplate,
		"generalspacepath_template":   cfg.Drivers.OCIS.GeneralSpacePathTemplate,
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
		"max_concurrency":             cfg.Drivers.S3NG.MaxConcurrency,
		"disable_versioning":          cfg.Drivers.S3NG.DisableVersioning,
		"lock_cycle_duration_factor":  cfg.Drivers.S3NG.LockCycleDurationFactor,
		"filemetadatacache": map[string]interface{}{
			"cache_store":               cfg.FilemetadataCache.Store,
			"cache_nodes":               cfg.FilemetadataCache.Nodes,
			"cache_database":            cfg.FilemetadataCache.Database,
			"cache_ttl":                 cfg.FilemetadataCache.TTL,
			"cache_disable_persistence": cfg.FilemetadataCache.DisablePersistence,
			"cache_auth_username":       cfg.FilemetadataCache.AuthUsername,
			"cache_auth_password":       cfg.FilemetadataCache.AuthPassword,
		},
		"idcache": map[string]interface{}{
			"cache_store":               cfg.IDCache.Store,
			"cache_nodes":               cfg.IDCache.Nodes,
			"cache_database":            cfg.IDCache.Database,
			"cache_ttl":                 cfg.IDCache.TTL,
			"cache_disable_persistence": cfg.IDCache.DisablePersistence,
			"cache_auth_username":       cfg.IDCache.AuthUsername,
			"cache_auth_password":       cfg.IDCache.AuthPassword,
		},
	}
}
