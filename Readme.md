# tcpblower
`tcpblower` can forward data between different ports and display it in a hex table format, making it useful for debugging embedded devices. It also supports multiple architectures.
## usage
```shell
$ tcpblower --port 9000 --peer-port 90001
```
## parameters
```
tcpblower can forward data between different ports and display it in a hex table format, making it useful for debugging embedded devices. It also supports multiple architectures.

Usage:
  tcpblower [flags]

Flags:
  -h, --help            help for tcpblower
  -P, --peer-port int   peer port to connect (default 34051)
  -p, --port int        port to listen (default 34050)
```