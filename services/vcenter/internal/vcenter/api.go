package vcenter

import "context"

type InventoryItem struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	MoRef string `json:"moRef"`
	Path  string `json:"path"`
}

type VMInfo struct {
	Name       string `json:"name"`
	MoRef      string `json:"moRef"`
	PowerState string `json:"powerState"`
	CPU        int32  `json:"cpu"`
	MemoryMB   int64  `json:"memoryMB"`
}

type PerfPoint struct {
	TS    string  `json:"ts"`
	Value float64 `json:"value"`
}

type PerfSeries struct {
	Metric string      `json:"metric"`
	Points []PerfPoint `json:"points"`
}

type EventItem struct {
	TS          string `json:"ts"`
	Type        string `json:"type"`
	Message     string `json:"message"`
	EntityMoRef string `json:"entityMoRef,omitempty"`
}

type AlarmItem struct {
	AlarmID     string `json:"alarmId"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	EntityMoRef string `json:"entityMoRef,omitempty"`
}

type TaskRef struct {
	TaskMoRef string `json:"taskMoRef"`
	State     string `json:"state"`
}

type API interface {
	ListInventory(ctx context.Context, types []string, nameContains string, limit int, cursor string) ([]InventoryItem, string, error)
	GetVM(ctx context.Context, moRef string, path string) (VMInfo, error)

	QueryPerf(ctx context.Context, objectType string, moRef string, metrics []string, startTime string, endTime string, intervalSeconds int) ([]PerfSeries, error)
	ListEvents(ctx context.Context, startTime string, endTime string, moRef string, types []string, limit int, cursor string) ([]EventItem, string, error)
	ListAlarms(ctx context.Context, moRef string) ([]AlarmItem, error)

	PowerVM(ctx context.Context, vmMoRef string, action string) (TaskRef, error)
	SnapshotVM(ctx context.Context, vmMoRef string, action string, name string, description string) (any, error)
}
