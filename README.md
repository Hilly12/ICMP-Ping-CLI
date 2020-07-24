# Ping Command Line Interface
Simulates a Ping to a host by continuously sending it ICMP echo requests.
<br />
To run:
<br />
`go run ping-cli.go [global options] [hostname / IP]`
<br />
After build:
<br />
`./ping-cli [global options] [hostname / IP]`
<br />
<br />
Global Options:
<br />
   --ttl value, -t value         time to live (default: "infinity")
   <br />
   --count value, -c value       number of pings (default: "infinity")
   <br />
   --packetsize value, -s value  number of bytes to be sent (default: 56)
   <br />
   --help, -h                    show help (default: false)
