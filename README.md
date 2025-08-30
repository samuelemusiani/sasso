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

## Proxmox Setup

Before sasso can fully function, you need to set up the Proxmox cluster in a
certain wa:
* Create a VXLAN Zone for sasso. The default name is `sasso` but it can be
  changed in the config file.
* Allow the VXLAN to function if the proxmox firewall is enable. To do this
  you could add an `IPSet` with all the proxmox nodes in the cluster and then
  add a cluster firewall rule from the IPSet to the same IPSet, protocol `udp`
  and destination port `4789`.
* Have a template for a VM to clone.
* Generate an API token with all the necessary permissions.

## Development

Sasso needs a PostgreSQL database. Use the `docker-compose-test.yml` file to
start a test database.
```bash
docker compose -f docker-compose-test.yml up -d
```
This docker compose also brings up a second databse for the router and an instace
of `lldap` as Sasso allows users to only authenticate with an external LDAP server.
To connect to to the UI of `lldap` navigate to `localhost:17170`. For the admin
password look inside the docker compose file.

To run the server with the default config:
```bash
go run ./server ./server/config/config.toml  
```
In the default config the Proxmox server is hosted on `localhost:8006`. You
can forward the Proxmox API to your localhost with:
```bash
ssh -L 8006:localhost:8006 <user>@<proxmox-ip>
```

To run the frontend you must enter the `frontend` directory and run:
```bash
npm run dev
```

To run the `sasso-router` service you can use:
```bash
go run ./router ./router/config/config.toml  
```
