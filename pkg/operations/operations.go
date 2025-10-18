package operations

type OperationType int

const (
	InstallOperation OperationType = iota
	RepairOperation
	UpdateOperation
)
