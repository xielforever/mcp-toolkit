package vcenter

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

type GovmomiClient struct {
	raw *govmomi.Client
}

func NewGovmomiClient(ctx context.Context, vcURL string, username string, password string, insecure bool, caFile string) (*GovmomiClient, error) {
	u, err := url.Parse(vcURL)
	if err != nil {
		return nil, err
	}
	u.User = url.UserPassword(username, password)

	c, err := govmomi.NewClient(ctx, u, insecure)
	if err != nil {
		return nil, err
	}
	_ = caFile
	return &GovmomiClient{raw: c}, nil
}

func (c *GovmomiClient) ListInventory(ctx context.Context, kinds []string, nameContains string, limit int, cursor string) ([]InventoryItem, string, error) {
	_ = cursor

	v := view.NewManager(c.raw.Client)

	vimTypes := make([]string, 0, len(kinds))
	for _, t := range kinds {
		switch strings.ToLower(t) {
		case "datacenter":
			vimTypes = append(vimTypes, "Datacenter")
		case "cluster":
			vimTypes = append(vimTypes, "ClusterComputeResource")
		case "host":
			vimTypes = append(vimTypes, "HostSystem")
		case "datastore":
			vimTypes = append(vimTypes, "Datastore")
		case "network":
			vimTypes = append(vimTypes, "Network")
		case "vm":
			vimTypes = append(vimTypes, "VirtualMachine")
		}
	}

	if len(vimTypes) == 0 {
		return []InventoryItem{}, "", nil
	}

	cv, err := v.CreateContainerView(ctx, c.raw.ServiceContent.RootFolder, vimTypes, true)
	if err != nil {
		return nil, "", err
	}
	defer cv.Destroy(ctx)

	var mos []mo.ManagedEntity
	pc := property.DefaultCollector(c.raw.Client)
	refs := []types.ManagedObjectReference{cv.Reference()}
	if err := pc.Retrieve(ctx, refs, []string{"name"}, &mos); err != nil {
		return nil, "", err
	}

	items := make([]InventoryItem, 0, len(mos))
	for _, e := range mos {
		ref := e.Reference()
		itemType := strings.ToLower(ref.Type)
		itemType = strings.TrimSuffix(itemType, "computeresource")
		itemType = strings.TrimSuffix(itemType, "system")
		itemType = strings.TrimSuffix(itemType, "machine")
		if itemType == "virtual" {
			itemType = "vm"
		}
		if itemType == "cluster" {
			itemType = "cluster"
		}
		if itemType == "host" {
			itemType = "host"
		}
		if itemType == "datastore" {
			itemType = "datastore"
		}
		if itemType == "datacenter" {
			itemType = "datacenter"
		}
		if itemType == "network" {
			itemType = "network"
		}

		if nameContains != "" && !strings.Contains(strings.ToLower(e.Name), strings.ToLower(nameContains)) {
			continue
		}

		items = append(items, InventoryItem{
			Type:  itemType,
			Name:  e.Name,
			MoRef: ref.Value,
			Path:  e.Name,
		})

		if limit > 0 && len(items) >= limit {
			break
		}
	}

	return items, "", nil
}

func (c *GovmomiClient) GetVM(ctx context.Context, moRef string, path string) (VMInfo, error) {
	_ = path
	vm := object.NewVirtualMachine(c.raw.Client, types.ManagedObjectReference{Type: "VirtualMachine", Value: moRef})
	var o mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), []string{"name", "runtime.powerState", "summary.config.numCpu", "summary.config.memorySizeMB"}, &o); err != nil {
		return VMInfo{}, err
	}

	return VMInfo{
		Name:       o.Name,
		MoRef:      moRef,
		PowerState: string(o.Runtime.PowerState),
		CPU:        o.Summary.Config.NumCpu,
			MemoryMB:   int64(o.Summary.Config.MemorySizeMB),
	}, nil
}

func (c *GovmomiClient) QueryPerf(ctx context.Context, objectType string, moRef string, metrics []string, startTime string, endTime string, intervalSeconds int) ([]PerfSeries, error) {
	_ = ctx
	_ = objectType
	_ = moRef
	_ = metrics
	_ = startTime
	_ = endTime
	_ = intervalSeconds
	return nil, errors.New("not implemented")
}

func (c *GovmomiClient) ListEvents(ctx context.Context, startTime string, endTime string, moRef string, types []string, limit int, cursor string) ([]EventItem, string, error) {
	_ = ctx
	_ = startTime
	_ = endTime
	_ = moRef
	_ = types
	_ = limit
	_ = cursor
	return nil, "", errors.New("not implemented")
}

func (c *GovmomiClient) ListAlarms(ctx context.Context, moRef string) ([]AlarmItem, error) {
	_ = ctx
	_ = moRef
	return nil, errors.New("not implemented")
}

func (c *GovmomiClient) PowerVM(ctx context.Context, vmMoRef string, action string) (TaskRef, error) {
	vm := object.NewVirtualMachine(c.raw.Client, types.ManagedObjectReference{Type: "VirtualMachine", Value: vmMoRef})
	switch strings.ToLower(action) {
	case "on":
		t, err := vm.PowerOn(ctx)
		if err != nil {
			return TaskRef{}, err
		}
		return TaskRef{TaskMoRef: t.Reference().Value, State: "submitted"}, nil
	case "off":
		t, err := vm.PowerOff(ctx)
		if err != nil {
			return TaskRef{}, err
		}
		return TaskRef{TaskMoRef: t.Reference().Value, State: "submitted"}, nil
	default:
		return TaskRef{}, errors.New("unsupported action")
	}
}

func (c *GovmomiClient) SnapshotVM(ctx context.Context, vmMoRef string, action string, name string, description string) (any, error) {
	_ = ctx
	_ = vmMoRef
	_ = action
	_ = name
	_ = description
	return nil, errors.New("not implemented")
}
