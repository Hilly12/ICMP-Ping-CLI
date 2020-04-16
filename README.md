# Ping CLI -
Simulates a Ping to a host by sending continuous ICSM echo requests to test its reachability.
To run:
`go run ping-cli.go [global options] [hostname / IP]`
After build:
`./ping-cli [global options] [hostname / IP]`

Global Options:
   --ttl value, -t value         time to live (default: "infinity")
   --count value, -c value       number of pings (default: "infinity")
   --packetsize value, -s value  number of bytes to be sent (default: 56)
   --help, -h                    show help (default: false)