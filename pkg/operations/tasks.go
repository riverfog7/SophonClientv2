package operations

import "SophonClientv2/internal/models"
import "SophonClientv2/internal/logging"

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
	case "repair":
		if req, ok := request.(models.RepairRequest); ok {
			return PerformRepair(req)
		}
	case "update":
		if req, ok := request.(models.UpdateRequest); ok {
			return PerformUpdate(req)
		}
	default:
		logging.GlobalLogger.Fatal("Unknown task type: " + taskType)
	}
	return models.TaskResponse{} // Should not reach here
}
