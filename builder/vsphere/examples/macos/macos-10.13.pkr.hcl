# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# source blocks are analogous to the "builders" in json templates. They are used
# in build blocks. A build block runs provisioners and post-processors on a
# source. Read the documentation for source blocks here:
# https://www.packer.io/docs/templates/hcl_templates/blocks/source
source "vsphere-iso" "example_osx" {
  CPUs         = 1
  RAM          = 4096
  boot_command = ["<enter><wait5>", "<leftCtrlOn><f2><leftCtrlOff>u<enter>t<enter><wait5>", "/Volumes/setup/setup.sh<enter>"]
  boot_wait    = "4m"
  cdrom_type   = "sata"
  configuration_parameters = {
    "ich7m.present" = "TRUE"
    "smc.present"   = "TRUE"
  }
  guest_os_type       = "darwin16_64Guest"
  host                = "esxi-mac.vsphere65.test"
  insecure_connection = "true"
  iso_checksum        = "file:///${path.root}/setup/out/sha256sums"
  iso_paths           = ["[datastore-mac] ISO/macOS 10.13.3.iso", "[datastore-mac] ISO/VMware Tools/10.2.0/darwin.iso"]
  iso_urls            = ["${path.root}/setup/out/setup.iso"]
  network_adapters {
    network_card = "e1000e"
  }
  password     = "jetbrains"
  ssh_password = "jetbrains"
  ssh_username = "jetbrains"
  storage {
    disk_size             = 32768
    disk_thin_provisioned = true
  }
  usb_controller = ["usb"]
  username       = "root"
  vcenter_server = "vcenter.vsphere65.test"
  vm_name        = "macos-packer"
}

# a build block invokes sources and runs provisioning steps on them. The
# documentation for build blocks can be found here:
# https://www.packer.io/docs/templates/hcl_templates/blocks/build
build {
  sources = ["source.vsphere-iso.example_osx"]

}
