# Sasso router service

To allow VNets to exit on the Internet, a router service is required. This
service is detached from the main Sasso server, but they communicate in order
to trigger events in both directions.

`sasso-router` is responsible of managing network subents. `sasso` will ask a new
subnet for each VNet and this sevice will respond with an available subnet and
the IP address of the router.

`sasso-router` is also responsible for managing the firewall configuration inside
the router. It will also be responbile for port forwarding and NAT rules.
