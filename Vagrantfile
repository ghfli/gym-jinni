# -*- mode: ruby -*-
# vi: set ft=ruby :

# All Vagrant configuration is done below. The "2" in Vagrant.configure
# configures the configuration version (we support older styles for
# backwards compatibility). Please don't change it unless you know what
# you're doing.
Vagrant.configure("2") do |config|
  # The most common configuration options are documented and commented below.
  # For a complete reference, please see the online documentation at
  # https://docs.vagrantup.com.

  # Every Vagrant development environment requires a box. You can search for
  # boxes at https://vagrantcloud.com/search.

  # Disable automatic box update checking. If you disable this, then
  # boxes will only be checked for updates when the user runs
  # `vagrant box outdated`. This is not recommended.
  # config.vm.box_check_update = false

  # Create a forwarded port mapping which allows access to a specific port
  # within the machine from a port on the host machine. In the example below,
  # accessing "localhost:8080" will access port 80 on the guest machine.
  # NOTE: This will enable public access to the opened port
  # config.vm.network "forwarded_port", guest: 80, host: 8080

  # Create a forwarded port mapping which allows access to a specific port
  # within the machine from a port on the host machine and only allow access
  # via 127.0.0.1 to disable public access
  # config.vm.network "forwarded_port", guest: 80, host: 8080, host_ip: "127.0.0.1"

  # Create a private network, which allows host-only access to the machine
  # using a specific IP.
  # config.vm.network "private_network", ip: "192.168.33.10"

  # Create a public network, which generally matched to bridged network.
  # Bridged networks make the machine appear as another physical device on
  # your network.
  # config.vm.network "public_network"

  # Share an additional folder to the guest VM. The first argument is
  # the path on the host to the actual folder. The second argument is
  # the path on the guest to mount the folder. And the optional third
  # argument is a set of non-required options.
  # config.vm.synced_folder "../data", "/vagrant_data"

  # Provider-specific configuration so you can fine-tune various
  # backing providers for Vagrant. These expose provider-specific options.
  # Example for VirtualBox:
  #
  # config.vm.provider "virtualbox" do |vb|
  #   # Display the VirtualBox GUI when booting the machine
  #   vb.gui = true
  #
  #   # Customize the amount of memory on the VM:
  #   vb.memory = "1024"
  # end
  #
  # View the documentation for the provider you are using for more
  # information on available options.

  # Enable provisioning with a shell script. Additional provisioners such as
  # Ansible, Chef, Docker, Puppet and Salt are also available. Please see the
  # documentation for more information about their specific syntax and use.
  # config.vm.provision "shell", inline: <<-SHELL
  #   apt-get update
  #   apt-get install -y apache2
  # SHELL

  config.vm.define :alx do |alx|
    alx.vm.box = "archlinux/archlinux"

    # See details at https://github.com/vagrant-libvirt/vagrant-libvirt.
    alx.vm.provider :libvirt do |dom, override|
      dom.cpus = 2          # default to 1
      dom.memory = "2048"   # default to 512M
      # Pass through /dev/random from the host to the VM
      dom.random :model => 'random'

      override.vm.synced_folder ".", "/vagrant", type: "nfs", nfs_udp: false, nfs_version: 4
    end

    alx.vm.provider "virtualbox" do |vb, override|
      # Display the VirtualBox GUI when booting the machine
      vb.gui = true
      vb.cpus = 2
      vb.memory = "2048"

      override.vm.synced_folder ".", "/vagrant", type: "virtualbox"
    end

    alx.vm.network "forwarded_port", guest: 22, host: 2222
    alx.vm.network "forwarded_port", guest: 5432, host: 5432
    alx.vm.network "forwarded_port", guest: 8080, host: 8080
    alx.vm.network "forwarded_port", guest: 8081, host: 8081
    alx.vm.network "forwarded_port", guest: 10000, host: 10000

    alx.vm.provision "setup-alx", type: "shell", path: "setup-alx.sh"
    # alx.vm.provision "file", source: "~/.gitconfig", destination: ".gitconfig"
  end
end
