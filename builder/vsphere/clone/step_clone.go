// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type CloneConfig,vAppConfig

package clone

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
	"github.com/hashicorp/packer-plugin-vsphere/builder/vsphere/common"
	"github.com/hashicorp/packer-plugin-vsphere/builder/vsphere/driver"
)

type vAppConfig struct {
	// Set values for the available vApp Properties to supply configuration parameters to a virtual machine cloned from
	// a template that came from an imported OVF or OVA file.
	//
	// -> **Note:** The only supported usage path for vApp properties is for existing user-configurable keys.
	// These generally come from an existing template that was created from an imported OVF or OVA file.
	// You cannot set values for vApp properties on virtual machines created from scratch,
	// virtual machines lacking a vApp configuration, or on property keys that do not exist.
	Properties map[string]string `mapstructure:"properties"`
}

type CloneConfig struct {
	// Name of source virtual machine. Path is optional.
	Template string `mapstructure:"template"`
	// The size of the disk in MiB.
	DiskSize int64 `mapstructure:"disk_size"`
	// Create the virtual machine as a linked clone from latest snapshot. Defaults to `false`.
	LinkedClone bool `mapstructure:"linked_clone"`
	// Set the network in which the VM will be connected to. If no network is
	// specified, `host` must be specified to allow Packer to look for the
	// available network. If the network is inside a network folder in vSphere inventory,
	// you need to provide the full path to the network.
	Network string `mapstructure:"network"`
	// Sets a custom MAC address to the network adapter. If set, the [network](#network) must be also specified.
	MacAddress string `mapstructure:"mac_address"`
	// VM notes.
	Notes string `mapstructure:"notes"`
	// If set to true, the virtual machine will be destroyed after the build completes.
	Destroy bool `mapstructure:"destroy"`
	// Set the vApp Options on the virtual machine image.
	// See the [vApp Options Configuration](/packer/plugins/builders/vmware/vsphere-clone#vapp-options-configuration)
	// section for more information.
	VAppConfig    vAppConfig           `mapstructure:"vapp"`
	StorageConfig common.StorageConfig `mapstructure:",squash"`
}

func (c *CloneConfig) Prepare() []error {
	var errs []error
	errs = append(errs, c.StorageConfig.Prepare()...)

	if c.Template == "" {
		errs = append(errs, fmt.Errorf("'template' is required"))
	}

	if c.LinkedClone == true && c.DiskSize != 0 {
		errs = append(errs, fmt.Errorf("'linked_clone' and 'disk_size' cannot be used together"))
	}

	if c.MacAddress != "" && c.Network == "" {
		errs = append(errs, fmt.Errorf("'network' is required when 'mac_address' is specified"))
	}

	return errs
}

type StepCloneVM struct {
	Config        *CloneConfig
	Location      *common.LocationConfig
	Force         bool
	GeneratedData *packerbuilderdata.GeneratedData
}

func (s *StepCloneVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	d := state.Get("driver").(driver.Driver)
	vmPath := path.Join(s.Location.Folder, s.Location.VMName)

	err := d.PreCleanVM(ui, vmPath, s.Force, s.Location.Cluster, s.Location.Host, s.Location.ResourcePool)
	if err != nil {
		state.Put("error", err)
		return multistep.ActionHalt
	}

	ui.Say("Cloning VM...")
	template, err := d.FindVM(s.Config.Template)
	if err != nil {
		state.Put("error", fmt.Errorf("Error finding vm to clone: %s", err))
		return multistep.ActionHalt
	}

	var disks []driver.Disk
	for _, disk := range s.Config.StorageConfig.Storage {
		disks = append(disks, driver.Disk{
			DiskSize:            disk.DiskSize,
			DiskEagerlyScrub:    disk.DiskEagerlyScrub,
			DiskThinProvisioned: disk.DiskThinProvisioned,
			ControllerIndex:     disk.DiskControllerIndex,
		})
	}

	vm, err := template.Clone(ctx, &driver.CloneConfig{
		Name:            s.Location.VMName,
		Folder:          s.Location.Folder,
		Cluster:         s.Location.Cluster,
		Host:            s.Location.Host,
		ResourcePool:    s.Location.ResourcePool,
		Datastore:       s.Location.Datastore,
		LinkedClone:     s.Config.LinkedClone,
		Network:         s.Config.Network,
		MacAddress:      strings.ToLower(s.Config.MacAddress),
		Annotation:      s.Config.Notes,
		VAppProperties:  s.Config.VAppConfig.Properties,
		PrimaryDiskSize: s.Config.DiskSize,
		StorageConfig: driver.StorageConfig{
			DiskControllerType: s.Config.StorageConfig.DiskControllerType,
			Storage:            disks,
		},
	})
	if err != nil {
		state.Put("error", err)
		return multistep.ActionHalt
	}
	if vm == nil {
		return multistep.ActionHalt
	}
	if s.Config.Destroy {
		state.Put("destroy_vm", s.Config.Destroy)
	}
	state.Put("vm", vm)
	return multistep.ActionContinue
}

func (s *StepCloneVM) Cleanup(state multistep.StateBag) {
	common.CleanupVM(state)
}
