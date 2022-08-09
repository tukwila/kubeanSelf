# -*- mode: ruby -*-
Vagrant.configure("2") do |config|
    config.vm.provider "virtualbox" do |v|
    v.memory = 2048
    end
    
    config.vm.define "default" do |default|
    default.vm.box = "Kiowa/kubean-e2e-vm-template"
    default.vm.box_version = "0"
    default.vm.network "public_network", ip: "default_ip", bridge: "ens192"
    default.vm.hostname="default"
    end
    
    config.vm.define "default2" do |default2|
    default2.vm.box = "Kiowa/kubean-e2e-vm-template"
    default2.vm.box_version = "0"
    default2.vm.network "public_network", ip: "default2_ip", bridge: "ens192"
    default2.vm.hostname="default2"
    end
    
end