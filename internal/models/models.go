package models

type GameOperationRequest struct {
	GameDir  string `json:"gamedir" validate:"required"`
	GameType string `json:"game_type" validate:"oneof=hk4e nap hkrpg"` // hkrpg not implemented in python
	TempDir  string `json:"tempdir,omitempty"`
}

type InstallRequest struct {
	GameOperationRequest
	InstallRelType string `json:"install_reltype" validate:"oneof=os cn"`
}

type UpdateRequest struct {
	GameOperationRequest
	Predownload bool `json:"predownload"`
}

type RepairRequest struct {
	GameOperationRequest
	RepairMode string `json:"repair_mode" validate:"oneof=quick reliable"`
}

type TaskResponse struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type TaskStatus struct {
	TaskID   string   `json:"task_id"`
	Status   string   `json:"status" validate:"oneof=running completed failed cancelled pending"`
	Progress *float64 `json:"progress,omitempty"`
	Error    *string  `json:"error,omitempty"`
}

type OnlineGameInfo struct {
	GameType           string   `json:"game_type" validate:"oneof=hk4e nap hkrpg''"`
	Version            string   `json:"version"`
	InstallSize        int64    `json:"install_size"`
	UpdatableVersions  []string `json:"updatable_versions"`
	ReleaseType        string   `json:"release_type"`
	PreDownload        bool     `json:"pre_download"`
	PreDownloadVersion *string  `json:"pre_download_version,omitempty"`
	Error              *string  `json:"error,omitempty"`
}
