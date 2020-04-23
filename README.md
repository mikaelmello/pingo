# pingo

**pingo** is, currently, a simple CLI interface with the intent to test the reachability of a host on an IP network. Quite similar to [ping](https://en.wikipedia.org/wiki/Ping_(networking_utility)).

Between **cpping** and **pingo**, I had to learn Go to claim the latter. Learning the best ways to use the language's features everytime I tried one and it didn't quite work was a great experience (and I'm quite honest here).

It supports some features that **ping** supports, such as a custom timeout for the first `ECHO_REQUEST`, custom interval between `ECHO_REQUEST`s, a maximum amount of running time of the application, a custom TTL for packets and setting the max amount of `ECHO_REQUESTS` that can be sent before stopping.

It also supports both IPv4 and IPv6 addresses, along with hostnames.

## Installation

To install the program executable execute:

``` sh
$ go get github.com/mikaelmello/pingo/...
$ $GOPATH/bin/pingo # to execute the program
```

To manually build it, clone the repository and then build it

``` sh
$ git clone https://github.com/mikaelmello/pingo.git
$ cd pingo
$ make install
$ make build
$ ./pingo # to execute the program
```

## CLI Usage

```
Usage:
  pingo [hostname or ip address] [flags]

Flags:
  -c, --count int        Stop after sending count ECHO_REQUEST packets. With deadline option, ping waits for count
                         ECHO_REPLY packets, until the timeout expires. (default -1)

  -w, --deadline int     Specify a timeout, in seconds, before ping exits regardless of how many packets have been sent
                         or received. In this case ping does not stop after count packet are sent, it waits either for
                         deadline expire or until count probes are answered or for some error notification from network.
                         (default -1)

  -h, --help             help for pingo

  -i, --interval float   Wait interval seconds between sending each packet. The default is to wait for one second
                         between each packet normally. (default 1)

      --log-level int    Logging level, goes from top priority 0 (Panic) to lowest priority 6 (Trace). Values out of
                         this range log everything.

  -p, --privileged       Whether to use privileged mode. If yes, privileged raw ICMP endpoints are used, non-privileged
                         datagram-oriented otherwise. On Linux, to run unprivileged you must enable the setting 'sudo
                         sysctl -w net.ipv4.ping_group_range="0   2147483647"'. In order to run as a privileged user,
                         you can either run as sudo or execute 'setcap cap_net_raw=+ep <bin path>' to the path of the
                         binary. On Windows, you must run as privileged.

  -W, --timeout int      Time to wait for a response, in seconds. The option affects only timeout in absence of any
                         responses, otherwise ping waits for two RTTs. (default 10)

  -t, --ttl int          Set the IP Time to Live. (default 64)
```

## Package Usage

Soon ™

Things are already quite decoupled at the moment, the probably necessary adaptations would not require a lot of work.

## Privileged vs Non-privileged

This program uses raw sockets to make the ICMP echo requests and you probably need root permissions to receive or send raw sockets.

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

**Warning:** If you use the non-privileged mode, it is not possible to receive Time Exceeded ICMP messages, meaning that
a request that exceeded its time to live will never receive a response, timing out instead.

## Next steps

- Refactor and use more interfaces, so that we can improve our testing, maybe DI?
- Remove our tests' dependency on net packages for better mocking, maybe DI [2]?
- Add support to be used as a package

## Examples

``` sh
$ ./pingo cloudflare.com

PING cloudflare.com. (104.17.175.85:0) 24 bytes of data
24 bytes from cloudflare.com. (104.17.175.85:0): icmp_seq=1 ttl=51 time=156.164ms
24 bytes from cloudflare.com. (104.17.175.85:0): icmp_seq=2 ttl=51 time=155.883ms
24 bytes from cloudflare.com. (104.17.175.85:0): icmp_seq=3 ttl=51 time=153.782ms
24 bytes from cloudflare.com. (104.17.175.85:0): icmp_seq=4 ttl=51 time=154.843ms
^C
--- cloudflare.com. ping statistics ---
4 packets transmitted, 4 received, 0% packet loss, time 4.022s
rtt min/avg/max/mdev = 153.783/155.169/156.165/0.939 ms
```

```sh
$ sudo ./pingo cloudflare.com -c 2 -t 10 --privileged

PING cloudflare.com. (104.17.176.85) 24 bytes of data
From [random ip]: icmp_seq=1 time to live exceeded
From [random ip]: icmp_seq=2 time to live exceeded

--- cloudflare.com. ping statistics ---
2 packets transmitted, 0 received, 100% packet loss, time 1.53s
rtt min/avg/max/mdev = 0.000/0.000/0.000/0.000 ms
```

``` sh
$ ./pingo localhost -c 4

PING localhost. (127.0.0.1) 24 bytes of data
24 bytes from localhost. (127.0.0.1): icmp_seq=1 ttl=64 time=289µs
24 bytes from localhost. (127.0.0.1): icmp_seq=2 ttl=64 time=312µs
24 bytes from localhost. (127.0.0.1): icmp_seq=3 ttl=64 time=266µs
24 bytes from localhost. (127.0.0.1): icmp_seq=4 ttl=64 time=279µs

--- localhost. ping statistics ---
4 packets transmitted, 4 received, 0% packet loss, time 3.202s
rtt min/avg/max/mdev = 0.266/0.287/0.313/0.017 ms
```

``` sh
$ ./pingo example.com --log-level 3 -c 1

WARN[0000] You are running as non-privileged, meaning that it is not possible to receive TimeExceeded ICMP messages. Echo requests that exceed the configured TTL of 64 will be treated as timed out 
PING example.com. (93.184.216.34:0) 24 bytes of data
24 bytes from example.com. (93.184.216.34:0): icmp_seq=1 ttl=54 time=135.416ms

--- example.com. ping statistics ---
1 packets transmitted, 1 received, 0% packet loss, time 335ms
rtt min/avg/max/mdev = 135.416/135.416/135.416/0.000 ms
```
