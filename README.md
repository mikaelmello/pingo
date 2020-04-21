# pingo

Between **cpping** and **pingo**, I had to learn Go to use the latter.

**pingo** is, currently, a simple CLI interface with the intent to test the reachability of a host on an IP network. Quite similar to [ping](https://en.wikipedia.org/wiki/Ping_(networking_utility)).

It supports some features that **ping** supports, such as a custom timeout for the first `ECHO_REQUEST`, custom interval between `ECHO_REQUEST`s, a maximum amount of running time of the application, a custom TTL for packets and setting the max amount of `ECHO_REQUESTS` that can be sent before stopping.

It also supports both IPv4 and IPv6 addresses, along with hostnames.

The goal now is to make the core package more flexible and expose an API to be used as a package in other Go projects.

## CLI Usage

```
Usage:
  pingo [hostname or ip address] [flags]

Flags:
  -c, --count int      Stop after sending count ECHO_REQUEST packets. With deadline option, ping waits for count ECHO_REPLY packets, until the timeout expires. (default -1)
  
  -w, --deadline int   Specify a timeout, in seconds, before ping exits regardless of how many packets have been sent or received. In this case ping does not stop after count packet are sent, it waits either for deadline expire or until count probes are answered or for some error notification from network. (default 10)
  
  -h, --help           help for pingo
  
  -i, --interval int   Wait interval seconds between sending each packet. The default is to wait for one second between each packet normally. (default 1)
  
  -m, --pretty-print   Enables the pretty print of the results. False means a mode similar to the common ping.
  
  -p, --privileged     Whether to use privileged mode. If yes, privileged raw ICMP endpoints are used, non-privileged datagram-oriented otherwise. On Linux, to run unprivileged you must enable the setting 'sudo sysctl -w net.ipv4.ping_group_range="0   2147483647"'. In order to run as a privileged user, you can either run as sudo or execute 'setcap cap_net_raw=+ep <bin path>' to the path of the binary. On Windows, you must run as privileged.
  
  -W, --timeout int    Time to wait for a response, in seconds. The option affects only timeout in absence of any responses, otherwise ping waits for two RTTs. (default 10)
  
  -t, --ttl int        Set the IP Time to Live. (default 64)
  
      --verbose        Whether to enable verbose logs.

```

## Privileged vs Non-privileged

This program uses raw sockets to make the ICMP requests and you probably need root permissions to receive or send raw sockets.

Because of that, the default is to use a non-privileged and datagram-oriented mode, the networks "udp4" or "udp6", allowing us to send a few limited ICMP messages, exactly the echo and reply we want.

On Linux, you can enable it by setting

```
sudo sysctl -w net.ipv4.ping_group_range="0   2147483647"
```

or you can also run the CLI in the privileged mode, either as a privileged user or using setcap to allow your binary to bind raw sockets.

```
setcap CAP_NET_RAW+ep [binary path]
```

On Windows, you must use the privileged mode.