Let me check your existing setup scripts to understand your current approach:

[2 tools called]

Perfect! I can see you already have a Vagrant-based setup for Arch Linux. Your current `Vagrantfile` supports both **libvirt** (Linux) and **VirtualBox** (cross-platform). Let me provide you with a comprehensive guide to set up an Arch Linux VM across different platforms.

## Current Setup Analysis

Your existing setup already supports:
- ✅ **Linux** (via libvirt provider)
- ✅ **Windows/MacOS/Linux** (via VirtualBox provider)
- ✅ Automated provisioning via `setup-alx.sh`

## Cross-Platform Setup Guide

### Prerequisites by Platform

#### **Windows**
```powershell
# Install Vagrant
choco install vagrant

# Install VirtualBox
choco install virtualbox

# Or download from:
# - Vagrant: https://www.vagrantup.com/downloads
# - VirtualBox: https://www.virtualbox.org/wiki/Downloads
```

#### **MacOS**
```bash
# Install Vagrant and VirtualBox via Homebrew
brew install --cask vagrant
brew install --cask virtualbox

# Or download from official sites
```

#### **Linux (Ubuntu/Debian)**
```bash
# Install Vagrant
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install vagrant

# Option 1: VirtualBox (cross-platform compatible)
sudo apt install virtualbox

# Option 2: libvirt (Linux-native, better performance)
sudo apt install qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils
vagrant plugin install vagrant-libvirt
```

### Usage Commands (Cross-Platform)

```bash
# Navigate to project directory
cd /path/to/gym-jinni

# Start VM (automatically uses appropriate provider)
vagrant up alx

# SSH into VM
vagrant ssh alx

# Reload VM with provisioning (re-run setup-alx.sh)
vagrant reload alx --provision

# Stop VM
vagrant halt alx

# Destroy VM
vagrant destroy alx

# Check VM status
vagrant status
```

### Enhanced Cross-Platform Vagrantfile

Here's an improved version with better cross-platform support:

```ruby
# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.define :alx do |alx|
    alx.vm.box = "archlinux/archlinux"
    alx.vm.hostname = "gym-jinni-dev"

    # Port forwarding (works on all platforms)
    alx.vm.network "forwarded_port", guest: 22, host: 2222, id: "ssh"
    alx.vm.network "forwarded_port", guest: 5432, host: 5432, id: "postgres"
    alx.vm.network "forwarded_port", guest: 8080, host: 8080, id: "http"
    alx.vm.network "forwarded_port", guest: 8081, host: 8081, id: "grpc"
    alx.vm.network "forwarded_port", guest: 10000, host: 10000, id: "custom"

    # Linux: libvirt provider (best performance on Linux)
    alx.vm.provider :libvirt do |dom, override|
      dom.cpus = 2
      dom.memory = "2048"
      dom.random :model => 'random'
      override.vm.synced_folder ".", "/vagrant", type: "nfs", nfs_udp: false, nfs_version: 4
    end

    # Windows/MacOS/Linux: VirtualBox provider (universal compatibility)
    alx.vm.provider "virtualbox" do |vb, override|
      vb.name = "gym-jinni-archlinux"
      vb.gui = true
      vb.cpus = 2
      vb.memory = "2048"
      
      # Better performance settings
      vb.customize ["modifyvm", :id, "--natdnshostresolver1", "on"]
      vb.customize ["modifyvm", :id, "--natdnsproxy1", "on"]
      vb.customize ["modifyvm", :id, "--ioapic", "on"]
      
      override.vm.synced_folder ".", "/vagrant", type: "virtualbox"
    end

    # MacOS: VMware Fusion (optional, requires license)
    alx.vm.provider "vmware_fusion" do |v, override|
      v.vmx["memsize"] = "2048"
      v.vmx["numvcpus"] = "2"
      override.vm.synced_folder ".", "/vagrant", type: "vmware"
    end

    # Windows: Hyper-V (optional, Windows Pro/Enterprise only)
    alx.vm.provider "hyperv" do |h, override|
      h.cpus = 2
      h.memory = "2048"
      h.enable_virtualization_extensions = true
      override.vm.synced_folder ".", "/vagrant", type: "smb"
    end

    # Provisioning script
    alx.vm.provision "setup-alx", type: "shell", path: "setup-alx.sh"
    
    # Optional: Copy git config
    # alx.vm.provision "file", source: "~/.gitconfig", destination: ".gitconfig"
  end
end
```

### Platform-Specific Wrapper Scripts

#### **Windows PowerShell Script** (`setup-vm.ps1`)
```powershell
#!/usr/bin/env pwsh
# Setup Arch Linux VM on Windows

Write-Host "Setting up Arch Linux VM for gym-jinni development..."

# Check if Vagrant is installed
if (!(Get-Command vagrant -ErrorAction SilentlyContinue)) {
    Write-Host "Vagrant not found. Installing..."
    choco install vagrant -y
}

# Check if VirtualBox is installed
if (!(Get-Command VBoxManage -ErrorAction SilentlyContinue)) {
    Write-Host "VirtualBox not found. Installing..."
    choco install virtualbox -y
}

# Start VM
Write-Host "Starting Arch Linux VM..."
vagrant up alx

Write-Host "Done! SSH into VM with: vagrant ssh alx"
```

#### **MacOS/Linux Bash Script** (`setup-vm.sh`)
```bash
#!/bin/bash
# Setup Arch Linux VM on MacOS/Linux

set -e

echo "Setting up Arch Linux VM for gym-jinni development..."

# Detect OS
OS="$(uname -s)"

case "${OS}" in
    Linux*)
        echo "Detected Linux"
        if ! command -v vagrant &> /dev/null; then
            echo "Installing Vagrant..."
            wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
            echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
            sudo apt update && sudo apt install vagrant
        fi
        
        # Prefer libvirt on Linux
        if ! vagrant plugin list | grep -q vagrant-libvirt; then
            echo "Installing vagrant-libvirt plugin..."
            sudo apt install -y qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils
            vagrant plugin install vagrant-libvirt
        fi
        ;;
        
    Darwin*)
        echo "Detected MacOS"
        if ! command -v vagrant &> /dev/null; then
            echo "Installing Vagrant..."
            brew install --cask vagrant
        fi
        
        if ! command -v VBoxManage &> /dev/null; then
            echo "Installing VirtualBox..."
            brew install --cask virtualbox
        fi
        ;;
        
    *)
        echo "Unknown OS: ${OS}"
        exit 1
        ;;
esac

# Start VM
echo "Starting Arch Linux VM..."
vagrant up alx

echo "Done! SSH into VM with: vagrant ssh alx"
```

### Quick Start Commands

```bash
# Clone and setup (all platforms)
git clone <your-repo>
cd gym-jinni

# Linux/MacOS
chmod +x setup-vm.sh
./setup-vm.sh

# Windows (PowerShell as Administrator)
Set-ExecutionPolicy Bypass -Scope Process
.\setup-vm.ps1

# Or manually
vagrant up alx
vagrant ssh alx
```

### Troubleshooting

**VirtualBox on Linux with Secure Boot:**
```bash
# Sign VirtualBox modules for Secure Boot
sudo /usr/lib/virtualbox/vboxdrv.sh setup
```

**Network issues on Windows:**
```powershell
# Reset network adapters in VirtualBox
VBoxManage list hostonlyifs
VBoxManage hostonlyif remove "VirtualBox Host-Only Ethernet Adapter"
```

**Performance tip:** On Linux, use libvirt instead of VirtualBox for better performance. Your Vagrantfile already supports both!

Your current setup is already well-designed for cross-platform use. The main thing needed is to ensure users have Vagrant + VirtualBox (or libvirt on Linux) installed, then `vagrant up alx` works everywhere!

Good catch! The official Arch Linux box has indeed been removed from Vagrant Cloud. Let me show you how to set up Arch Linux with libvirt directly.

## Solution 1: Use Alternative Vagrant Boxes

There are community-maintained Arch Linux boxes still available:

```ruby
# In your Vagrantfile, replace:
alx.vm.box = "archlinux/archlinux"

# With one of these alternatives:
alx.vm.box = "generic/arch"           # Most reliable, multi-provider
# OR
alx.vm.box = "ogarcia/archlinux-x64"  # Another option
```

## Solution 2: Create Arch Linux VM Directly with libvirt/virt-install

Here's a script to create an Arch Linux VM using libvirt directly without Vagrant:

### **`setup-archlinux-libvirt.sh`**

```bash
#!/bin/bash
set -e

VM_NAME="gym-jinni-arch"
VM_MEMORY=2048
VM_CPUS=2
VM_DISK_SIZE=20G
ISO_URL="https://mirror.rackspace.com/archlinux/iso/latest/archlinux-x86_64.iso"
ISO_PATH="/var/lib/libvirt/images/archlinux-latest.iso"
DISK_PATH="/var/lib/libvirt/images/${VM_NAME}.qcow2"

echo "Setting up Arch Linux VM with libvirt..."

# Check if libvirt is installed
if ! command -v virsh &> /dev/null; then
    echo "Installing libvirt..."
    sudo apt install -y qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils virt-manager virt-install
    sudo systemctl enable --now libvirtd
    sudo usermod -aG libvirt $USER
    echo "Please log out and back in for group changes to take effect"
fi

# Download Arch Linux ISO if not exists
if [ ! -f "$ISO_PATH" ]; then
    echo "Downloading Arch Linux ISO..."
    sudo curl -L "$ISO_URL" -o "$ISO_PATH"
fi

# Check if VM already exists
if virsh list --all | grep -q "$VM_NAME"; then
    echo "VM $VM_NAME already exists"
    read -p "Destroy and recreate? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        virsh destroy "$VM_NAME" 2>/dev/null || true
        virsh undefine "$VM_NAME" --remove-all-storage 2>/dev/null || true
    else
        echo "Exiting..."
        exit 0
    fi
fi

# Create VM
echo "Creating VM..."
sudo virt-install \
    --name "$VM_NAME" \
    --memory "$VM_MEMORY" \
    --vcpus "$VM_CPUS" \
    --disk path="$DISK_PATH",size=20,format=qcow2 \
    --cdrom "$ISO_PATH" \
    --os-variant archlinux \
    --network network=default \
    --graphics vnc,listen=0.0.0.0 \
    --noautoconsole \
    --boot uefi

echo "VM created! Connect with:"
echo "  virt-manager (GUI)"
echo "  or virsh console $VM_NAME"
echo ""
echo "After installation, you can:"
echo "  virsh start $VM_NAME"
echo "  virsh shutdown $VM_NAME"
echo "  virsh console $VM_NAME"
```

## Solution 3: Build Your Own Vagrant Box from Arch Linux ISO

### **`build-arch-box.sh`**

```bash
#!/bin/bash
# Build a custom Arch Linux Vagrant box for libvirt

set -e

BOX_NAME="archlinux-custom"
ISO_URL="https://mirror.rackspace.com/archlinux/iso/latest/archlinux-x86_64.iso"

echo "Building custom Arch Linux Vagrant box..."

# Install packer if not available
if ! command -v packer &> /dev/null; then
    echo "Installing Packer..."
    wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
    echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
    sudo apt update && sudo apt install packer
fi

# Create Packer template
cat > arch-box.pkr.hcl <<'EOF'
packer {
  required_plugins {
    qemu = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/qemu"
    }
  }
}

source "qemu" "archlinux" {
  iso_url           = "https://mirror.rackspace.com/archlinux/iso/latest/archlinux-x86_64.iso"
  iso_checksum      = "file:https://mirror.rackspace.com/archlinux/iso/latest/sha256sums.txt"
  output_directory  = "output-archlinux"
  shutdown_command  = "sudo systemctl poweroff"
  disk_size         = "20000"
  format            = "qcow2"
  accelerator       = "kvm"
  ssh_username      = "vagrant"
  ssh_password      = "vagrant"
  ssh_timeout       = "20m"
  vm_name           = "archlinux"
  net_device        = "virtio-net"
  disk_interface    = "virtio"
  boot_wait         = "5s"
  boot_command      = [
    "<enter><wait10><wait10><wait10>",
    "curl -O http://{{ .HTTPIP }}:{{ .HTTPPort }}/install.sh<enter><wait>",
    "bash install.sh<enter>"
  ]
  http_directory    = "http"
  headless          = true
}

build {
  sources = ["source.qemu.archlinux"]
}
EOF

echo "Packer template created. You'll need to create an install script."
echo "See: https://github.com/elasticdog/packer-arch for a complete example"
```

## Solution 4: Use Docker/Podman Instead (Recommended for Development)

For development purposes, you might consider using containers instead:

### **`docker-compose.yml`**

```yaml
version: '3.8'

services:
  archlinux-dev:
    image: archlinux:latest
    container_name: gym-jinni-arch
    hostname: gym-jinni-dev
    privileged: true
    volumes:
      - .:/workspace
      - /sys/fs/cgroup:/sys/fs/cgroup:ro
    ports:
      - "2222:22"
      - "5432:5432"
      - "8080:8080"
      - "8081:8081"
      - "10000:10000"
    working_dir: /workspace
    command: /bin/bash -c "pacman -Syu --noconfirm && /workspace/setup-alx.sh && tail -f /dev/null"
```

## **Recommended Approach: Updated Vagrantfile with Alternative Box**

Here's your updated `Vagrantfile`:

```ruby
# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.define :alx do |alx|
    # Use generic/arch instead of archlinux/archlinux
    alx.vm.box = "generic/arch"
    alx.vm.hostname = "gym-jinni-dev"

    # Linux: libvirt provider
    alx.vm.provider :libvirt do |dom, override|
      dom.cpus = 2
      dom.memory = "2048"
      dom.random :model => 'random'
      override.vm.synced_folder ".", "/vagrant", type: "nfs", nfs_udp: false, nfs_version: 4
    end

    # VirtualBox provider
    alx.vm.provider "virtualbox" do |vb, override|
      vb.name = "gym-jinni-archlinux"
      vb.gui = true
      vb.cpus = 2
      vb.memory = "2048"
      vb.customize ["modifyvm", :id, "--natdnshostresolver1", "on"]
      override.vm.synced_folder ".", "/vagrant", type: "virtualbox"
    end

    # Port forwarding
    alx.vm.network "forwarded_port", guest: 22, host: 2222, id: "ssh"
    alx.vm.network "forwarded_port", guest: 5432, host: 5432, id: "postgres"
    alx.vm.network "forwarded_port", guest: 8080, host: 8080, id: "http"
    alx.vm.network "forwarded_port", guest: 8081, host: 8081, id: "grpc"
    alx.vm.network "forwarded_port", guest: 10000, host: 10000, id: "custom"

    # Provisioning
    alx.vm.provision "setup-alx", type: "shell", path: "setup-alx.sh"
  end
end
```

## Quick Commands

```bash
# Using the updated Vagrantfile with generic/arch
vagrant up alx

# Or manually with libvirt (no Vagrant)
virsh list --all
virt-manager  # GUI tool

# Or with Docker
docker run -it --rm -v $(pwd):/workspace archlinux:latest bash
```

**Best recommendation:** Use `generic/arch` box in your Vagrantfile - it's actively maintained and works with both libvirt and VirtualBox. Just change that one line in your Vagrantfile and run `vagrant up alx` again!
