# netns

This project explores the [Linux network namespace](https://en.wikipedia.org/wiki/Linux_namespaces#Network_(net)), using tools from the [`iproute2`](https://en.wikipedia.org/wiki/Iproute2) utility.

In particular, it uses commands like:

* `ip netns` to manage network namespaces
* `ip link` to configure virtual network interfaces (i.e. `veth`)
* `ip addr` to add address range to new network interfaces
* `ip route` to manipulate route entries in the kernel routing table

For testing purposes, a TCP and UDP servers have been included in the `cmd` folder.

The following commands have been tested on Ubuntu 16.04.6 LTS.

**Note that `sudo` privileges are required.**

## Create the Virtual Network Interfaces (veth)

To create a pair of [`veth`](https://man7.org/linux/man-pages/man4/veth.4.html#:~:text=The%20veth%20devices%20are%20virtual,always%20created%20in%20interconnected%20pairs.) interfaces named `veth0` and `veth1` on the localhost:
```sh
ip link add veth0 type veth peer name veth1

ip link show veth0
134: veth0@veth1: <BROADCAST,MULTICAST> mtu 1500 qdisc noop state DOWN mode DEFAULT group default qlen 1000
    link/ether 0a:c7:eb:d4:ab:2a brd ff:ff:ff:ff:ff:ff

ip link show veth1
133: veth1@veth0: <BROADCAST,MULTICAST> mtu 1500 qdisc noop state DOWN mode DEFAULT group default qlen 1000
    link/ether fe:4a:e5:5b:52:fd brd ff:ff:ff:ff:ff:ff
```

Configure the `veth0` interface with IP address range 10.0.1.0/24:
```sh
ip addr add 10.0.1.0/24 dev veth0

ip link set veth0 up

ip addr show veth0
134: veth0@veth1: <BROADCAST,MULTICAST,M-DOWN> mtu 1500 qdisc noop state DOWN group default qlen 1000
    link/ether 0a:c7:eb:d4:ab:2a brd ff:ff:ff:ff:ff:ff
    inet 10.0.1.0/24 scope global veth0
       valid_lft forever preferred_lft forever
```

## Create a New Network Namespace

Create a new network namespace named `vnet`:
```sh
ip netns add vnet

ip netns show vnet
vnet
```

Move the `veth1` to the `vnet` network namespace:
```sh
ip link set veth1 netns vnet
```

Notice that `veth1` is no longer in the default host network namespace:
```sh
ip link show veth1
Device "veth1" does not exist.
```

It can be viewed in the `vnet` network namespace using the `ip netns exec` command:
```sh
ip netns exec vnet ip link show veth1
133: veth1@if134: <BROADCAST,MULTICAST> mtu 1500 qdisc noop state DOWN mode DEFAULT group default qlen 1000
    link/ether fe:4a:e5:5b:52:fd brd ff:ff:ff:ff:ff:ff link-netnsid 0
```

There are two parts to the above command:

1. `ip netns exec vnet` allows us to execute a command in the `vnet` network namespace
1. `ip link show veth1` is the command to be executed

> 💡 Tips: The `ip netns exec vnet <command>` command can be shortened to `ip -n vnet <command>`.

Configure the `veth1` interface by assigning it the IP address range 10.0.2.0/24:
```sh
ip netns exec vnet ip addr add 10.0.2.0/24 dev veth1

ip netns exec vnet ip link set veth1 up

ip netns exec vnet ip link show veth1
133: veth1@if134: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP mode DEFAULT group default qlen 1000
    link/ether fe:4a:e5:5b:52:fd brd ff:ff:ff:ff:ff:ff link-netnsid 0
```

_Notice that we deliberately assign different subnet address to the `veth` pair, so that we can explore routing later_

## Test with ICMP

Let's try to ping the `veth1` interface:
```sh
ping -c10 10.0.2.1
PING 10.0.2.1 (10.0.2.1) 56(84) bytes of data.
From 96.1.221.111 icmp_seq=8 Destination Host Unreachable

--- 10.0.2.1 ping statistics ---
10 packets transmitted, 0 received, +1 errors, 100% packet loss, time 9210ms
pipe 3
```

It doesn't work - let's use the `ip route` to do some diagnosis:
```sh
ip route get 10.0.2.1
10.0.2.1 via 192.168.1.254 dev wlp110s0  src 192.168.1.71
    cache
```

The packets are being routed to my `wlp110s0` interface, instead of the `veth0` interface.

Let's examine the route table:
```sh
ip route
default via 192.168.1.254 dev wlp110s0  proto static  metric 600
10.0.1.0/24 dev veth0  proto kernel  scope link  src 10.0.1.0
```
There are no route entries for the 10.0.2.0/24 range. Hence, the `default` route is used!

We need to add an entry for the 10.0.2.0/24 range, so that all packets destined for IP address in that range are routed to the `veth0` interface:
```sh
ip route add 10.0.2.0/24 dev veth0 scope link

ip route
default via 192.168.1.254 dev wlp110s0  proto static  metric 600
10.0.1.0/24 dev veth0  proto kernel  scope link  src 10.0.1.0
10.0.2.0/24 dev veth0  scope link

# great! looks like it's detecting the right interface to use
ip route get 10.0.2.0
10.0.2.0 dev veth0  src 10.0.1.0
    cache
```

Let's try the ping command again:
```
ping -c 10 10.0.2.0
PING 10.0.2.0 (10.0.2.0) 56(84) bytes of data.
From 10.0.1.0 icmp_seq=1 Destination Host Unreachable
From 10.0.1.0 icmp_seq=2 Destination Host Unreachable
From 10.0.1.0 icmp_seq=3 Destination Host Unreachable
From 10.0.1.0 icmp_seq=4 Destination Host Unreachable
From 10.0.1.0 icmp_seq=5 Destination Host Unreachable
From 10.0.1.0 icmp_seq=6 Destination Host Unreachable
From 10.0.1.0 icmp_seq=7 Destination Host Unreachable
From 10.0.1.0 icmp_seq=8 Destination Host Unreachable
From 10.0.1.0 icmp_seq=9 Destination Host Unreachable
From 10.0.1.0 icmp_seq=10 Destination Host Unreachable

--- 10.0.2.0 ping statistics ---
10 packets transmitted, 0 received, +10 errors, 100% packet loss, time 9217ms
pipe 4
```
Still no luck...

Let's investigate the route tables of the `vnet` network namespace:
```
ip netns exec vnet ip route
10.0.2.0/24 dev veth1  proto kernel  scope link  src 10.0.2.0
```
Ah.. so it's missing the route to the 10.0.1.0/24 IP address range.

Let's add it:
```
ip netns exec vnet ip route add 10.0.1.0 dev veth1 scope link

ip netns exec vnet ip route
10.0.1.0 dev veth1  scope link
10.0.2.0/24 dev veth1  proto kernel  scope link  src 10.0.2.0
```

Let's try the ping again:
```sh
ping -c 10 10.0.2.0
PING 10.0.2.0 (10.0.2.0) 56(84) bytes of data.
64 bytes from 10.0.2.0: icmp_seq=1 ttl=64 time=0.134 ms
64 bytes from 10.0.2.0: icmp_seq=2 ttl=64 time=0.074 ms
64 bytes from 10.0.2.0: icmp_seq=3 ttl=64 time=0.074 ms
64 bytes from 10.0.2.0: icmp_seq=4 ttl=64 time=0.074 ms
^C
--- 10.0.2.0 ping statistics ---
4 packets transmitted, 4 received, 0% packet loss, time 3072ms
rtt min/avg/max/mdev = 0.074/0.089/0.134/0.026 ms
```
And it works! 🎉🎉

We can use `tcpdump` to confirm that the packets are routed to the expected interfaces:
```sh
tcpdump -i veth0 icmp
tcpdump: verbose output suppressed, use -v or -vv for full protocol decode
listening on veth0, link-type EN10MB (Ethernet), capture size 262144 bytes
10:57:27.192552 IP 10.0.1.0 > 10.0.2.0: ICMP echo request, id 9557, seq 16, length 64
10:57:27.192595 IP 10.0.2.0 > 10.0.1.0: ICMP echo reply, id 9557, seq 16, length 64
10:57:28.220390 IP 10.0.1.0 > 10.0.2.0: ICMP echo request, id 9557, seq 17, length 64
10:57:28.220429 IP 10.0.2.0 > 10.0.1.0: ICMP echo reply, id 9557, seq 17, length 64
10:57:29.240463 IP 10.0.1.0 > 10.0.2.0: ICMP echo request, id 9557, seq 18, length 64
10:57:29.240506 IP 10.0.2.0 > 10.0.1.0: ICMP echo reply, id 9557, seq 18, length 64
10:57:30.264403 IP 10.0.1.0 > 10.0.2.0: ICMP echo request, id 9557, seq 19, length 64
10:57:30.264442 IP 10.0.2.0 > 10.0.1.0: ICMP echo reply, id 9557, seq 19, length 64
10:57:31.292395 IP 10.0.1.0 > 10.0.2.0: ICMP echo request, id 9557, seq 20, length 64
10:57:31.292440 IP 10.0.2.0 > 10.0.1.0: ICMP echo reply, id 9557, seq 20, length 64
10:57:32.312438 IP 10.0.1.0 > 10.0.2.0: ICMP echo request, id 9557, seq 21, length 64
10:57:32.312478 IP 10.0.2.0 > 10.0.1.0: ICMP echo reply, id 9557, seq 21, length 64

ip netns exec vnet tcpdump -i veth1 icmp -l
tcpdump: verbose output suppressed, use -v or -vv for full protocol decode
listening on veth1, link-type EN10MB (Ethernet), capture size 262144 bytes
^C10:57:49.720446 IP 10.0.1.0 > 10.0.2.0: ICMP echo request, id 9557, seq 38, length 64
10:57:49.720484 IP 10.0.2.0 > 10.0.1.0: ICMP echo reply, id 9557, seq 38, length 64
10:57:50.744444 IP 10.0.1.0 > 10.0.2.0: ICMP echo request, id 9557, seq 39, length 64
10:57:50.744483 IP 10.0.2.0 > 10.0.1.0: ICMP echo reply, id 9557, seq 39, length 64
10:57:51.768468 IP 10.0.1.0 > 10.0.2.0: ICMP echo request, id 9557, seq 40, length 64
10:57:51.768508 IP 10.0.2.0 > 10.0.1.0: ICMP echo reply, id 9557, seq 40, length 64
10:57:52.792426 IP 10.0.1.0 > 10.0.2.0: ICMP echo request, id 9557, seq 41, length 64
10:57:52.792473 IP 10.0.2.0 > 10.0.1.0: ICMP echo reply, id 9557, seq 41, length 64
10:57:53.816467 IP 10.0.1.0 > 10.0.2.0: ICMP echo request, id 9557, seq 42, length 64
10:57:53.816512 IP 10.0.2.0 > 10.0.1.0: ICMP echo reply, id 9557, seq 42, length 64
10:57:54.840403 IP 10.0.1.0 > 10.0.2.0: ICMP echo request, id 9557, seq 43, length 64
10:57:54.840418 IP 10.0.2.0 > 10.0.1.0: ICMP echo reply, id 9557, seq 43, length 64
10:57:55.864453 IP 10.0.1.0 > 10.0.2.0: ICMP echo request, id 9557, seq 44, length 64
```

## Test with TCP

Let's test with our TCP server in the `vnet` namespace:
```sh
ip netns exec vnet /usr/local/go/bin/go run ./cmd/tcp/...
2020/06/10 11:06:11 listening at 0.0.0.0:4078 (tcp)...
```

Open a TCP connection to it using `netcat` from the host network namespace, and send it a `hello` message:
```sh
echo "hello" | nc -4 -q1 10.0.2.0 4078
[2020-06-10 11:06:50] hello
```
The server responds with a message containing the original payload and a timestamp.

The server also logs the request that it receives:
```sh
ip netns exec vnet /usr/local/go/bin/go run ./cmd/tcp/...
2020/06/10 11:06:11 listening at 0.0.0.0:4078 (tcp)...
2020/06/10 11:06:50 received: "hello" (size_bytes=6)
```

## Examine the Network Namespace

To view the processes running in a network namespace, we can use the `ip netns pids` command:
```sh
ip netns pids vnet
14091
14179

ps aux | grep 14091
root     14091  0.0  0.0 1287368 18588 pts/2   Sl+  11:06   0:00 /usr/local/go/bin/go run ./cmd/tcp/...
```
