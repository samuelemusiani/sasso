# Sasso

## Proxmox Setup

Before sasso can fully function, you need to set up a Proxmox cluster in a
certain way.
* Create a VXLAN Zone for sasso. It must be named `sasso`.
* Allow the VXLAN to function if the proxmox firewall is enable. To do this
  you could add an `IPSet` with all the proxmox nodes in the cluster and then
  add a cluster firewall rule from the IPSet to the IPSet, protocol `udp` and
  destination port `4789`.
* Have a template for a VM to clone.

## Develop

Sasso needs a PostgreSQL database. Use the `docker-compose-test.yml` file to start a test database.
```bash
docker compose -f docker-compose-test.yml up -d
```

To run the server with the default config:
```bash
go run ./... ./server/config/config.toml  
```

To run the frontend you must enter the `frontend` directory and:
```bash
npm run dev
```
