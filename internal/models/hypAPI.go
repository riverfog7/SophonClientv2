package models

// getGameBranches API response model
type HYPGameInfo struct {
	ID  string `json:"id"`
	Biz string `json:"biz"`
}

type HYPGameCategory struct {
	CategoryId    string `json:"category_id"`
	MatchingField string `json:"matching_field"`
}

type HYPGameBranch struct {
	PackageId  string            `json:"package_id"`
	Branch     string            `json:"branch"`
	Password   string            `json:"password"`
	Tag        string            `json:"tag"`
	DiffTags   []string          `json:"diff_tags"`
	Categories []HYPGameCategory `json:"categories"`
}

type HYPGame struct {
	Game HYPGameInfo   `json:"game"`
	Main HYPGameBranch `json:"main"`
	// predownload branch is optional
	PreDownload *HYPGameBranch `json:"pre_download,omitempty"`
}

type HYPGetGameBranchesData struct {
	GameBranches []HYPGame `json:"game_branches"`
}

type HYPGetGameBranchesResponse struct {
	Retcode int                    `json:"retcode"`
	Message string                 `json:"message"`
	Data    HYPGetGameBranchesData `json:"data"`
}

// getGameConfigs API response model
type HYPGameLogExportFile struct {
	FileType string `json:"file_type"`
	Method   string `json:"method"`
	Path     string `json:"path"`
}

type HYPGameLogExportConfig struct {
	FileSizeFilter string                 `json:"file_size_filter"`
	ExportTimeout  string                 `json:"export_timeout"`
	ExportFiles    []HYPGameLogExportFile `json:"export_files"`
}

type HYPLaunchConfig struct {
	Game                          HYPGameInfo             `json:"game"`
	ExeFileName                   string                  `json:"exe_file_name"`
	InstallationDir               string                  `json:"installation_dir"`
	AudioPkgScanDir               string                  `json:"audio_pkg_scan_dir"`
	AudioPkgResDir                string                  `json:"audio_pkg_res_dir"`
	AudioPkgCacheDir              string                  `json:"audio_pkg_cache_dir"`
	GameCachedResDir              string                  `json:"game_cached_res_dir"`
	GameScreenshotDir             string                  `json:"game_screenshot_dir"`
	GameLogGenDir                 string                  `json:"game_log_gen_dir"`
	GameCrashFileGenDir           string                  `json:"game_crash_file_gen_dir"`
	DefaultDownloadMode           string                  `json:"default_download_mode"`
	EnableCustomerService         bool                    `json:"enable_customer_service"`
	LocalResDir                   string                  `json:"local_res_dir"`
	LocalResCacheDir              string                  `json:"local_res_cache_dir"`
	ResCategoryDir                string                  `json:"res_category_dir"`
	GameResCutDir                 string                  `json:"game_res_cut_dir"`
	EnableGameLogExport           bool                    `json:"enable_game_log_export"`
	GameLogExportConfig           *HYPGameLogExportConfig `json:"game_log_export_config"`
	BlacklistDir                  string                  `json:"blacklist_dir"`
	WpfExeDir                     string                  `json:"wpf_exe_dir"`
	WpfPkgVersionDir              string                  `json:"wpf_pkg_version_dir"`
	EnableAudioPkgMgmt            bool                    `json:"enable_audio_pkg_mgmt"`
	AudioPkgConfigDir             string                  `json:"audio_pkg_config_dir"`
	EnableResourceDeletionAdapter bool                    `json:"enable_resource_deletion_adapter"`
	EnableResourceBlacklist       bool                    `json:"enable_resource_blacklist"`
	EnableRedundantFileCleanup    bool                    `json:"enable_redundant_file_cleanup"`
	RedundantFileCleanupPaths     []string                `json:"redundant_file_cleanup_paths"`
	EnableV2GameDetection         bool                    `json:"enable_v2_game_detection"`
	RelatedProcesses              []string                `json:"related_processes"`
	EnableLdiff                   bool                    `json:"enable_ldiff"`
}

type HYPGetGameConfigsData struct {
	LaunchConfigs []HYPLaunchConfig `json:"launch_configs"`
}

type HYPGetGameConfigsResponse struct {
	Retcode int                   `json:"retcode"`
	Message string                `json:"message"`
	Data    HYPGetGameConfigsData `json:"data"`
}
