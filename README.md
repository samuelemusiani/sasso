# Sasso

Sasso is a VPS built on top of Proxmox. It allows users to create and manage
virtual machines in a controlled and secure environment, without giving them
direct access to the Proxmox cluster.

This service should be used when multiple users need to create and manage their
own virtual machines, but they don't have direct access to the Proxmox cluster
where the VMs are hosted.

Sasso provides resource management for each user through a web interface.
It creates every virtual machine in a separate Proxmox VNet using a VXLAN Zone
to keep the VMs from differente users isolated from each other.

> [!WARNING]
> At the moment Sasso is in an very early stage of development and is not
> fully functional. Expect bugs, missing features and breaking changes.

This service was developed to be used by [ADMStaff](https://students.cs.unibo.it).

Please refer to the [Wiki](https://github.com/samuelemusiani/sasso/wiki) for more
information about the architecture and how to deploy Sasso.
