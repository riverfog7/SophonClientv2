package operations

import "SophonClientv2/internal/models"

func PerformInstall(request models.InstallRequest) models.TaskResponse {
	// TODO: Implement install
	return models.TaskResponse{}
}

func PerformRepair(request models.RepairRequest) models.TaskResponse {
	// TODO: Implement repair
	return models.TaskResponse{}
}

func PerformUpdate(request models.UpdateRequest) models.TaskResponse {
	// TODO: Implement update
	return models.TaskResponse{}
}

func RunTask(taskType string, request interface{}) models.TaskResponse {
	switch taskType {
	case "install":
		if req, ok := request.(models.InstallRequest); ok {
			return PerformInstall(req)
		}
		return models.TaskResponse{Status: "failed", Message: "invalid request type for install task"}
	case "repair":
		if req, ok := request.(models.RepairRequest); ok {
			return PerformRepair(req)
		}
		return models.TaskResponse{Status: "failed", Message: "invalid request type for repair task"}
	case "update":
		if req, ok := request.(models.UpdateRequest); ok {
			return PerformUpdate(req)
		}
		return models.TaskResponse{Status: "failed", Message: "invalid request type for update task"}
	default:
		return models.TaskResponse{Status: "failed", Message: "unknown task type: " + taskType}
	}
}
