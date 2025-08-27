# Sasso router service

To allow VNets to exit on the Internet, a router service is required. This
service is detached from the main Sasso server, but they communicate in order
to trigger events in both directions.

`sasso-router` is responsible of managing network subents. `sasso` will ask a new
subnet for each VNet and this sevice will respond with an available subnet and
the IP address of the router.

`sasso-router` is also responsible for managing the firewall configuration inside
the router. It will also be responbile for port forwarding and NAT rules.

## Configurations

Ip forwarding must be enabled in the kernel. Add the following line to `/etc/sysctl.d/ip_forward.conf`
```
net.ipv4.ip_forward=1
```

## Firewall

For now only shorewall is supported. The default configuration is the following:
`/etc/shorewall/zones`
```
#ZONE	TYPE
fw	    firewall
out	    ip
```

`/etc/shorewall/interfaces`
```
#ZONE	INTERFACE
out	    eth0
```

`/etc/shorewall/interfaces`
```
#ZONE	INTERFACE
out	    eth0
```

`/etc/shorewall/policy`
```
#SOURCE		DEST	POLICY	LOG
fw		    all	    ACCEPT

out		    all	    DROP
```

`/etc/shorewall/rules`
```
#ACTION		    SOURCE	DEST	PROTO	DPORT	SPORT
Ping(ACCEPT)	all	    fw
ACCEPT		    out	    fw	    tcp	    ssh
```

`/etc/shorewall/snat`
```
#ACTION		SOURCE		    DEST
MASQUERADE	10.254.0.0/16	eth0
```

## Internal

When sasso add a new VNet, the steps are:
- Add a new zone (the name of the VNet)
- Add a new interface (type ip)
- Add a new policy (from the VNet to out ACCEPT)
- Add a new policy (from the VNet to all DROP)
