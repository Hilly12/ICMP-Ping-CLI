package main

import (
	"os"
	"net"
	"log"
	"time"
	"fmt"
	"errors"
	"github.com/urfave/cli"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	ipv4ProtocolNum = 1
	ipv6ProtocolNum = 58
	readDeadline = 2 // seconds
	defaultBytes = 56
)

/* Please be forgiving, just learnt go :) */
func main() {
	/* Initialize cli application with three flags -t, -c, -s */
	app := cli.NewApp()

	app.Name = "Ping CLI"
	app.Usage = "Tests reachability of a host by sending it ICMP echo requests"
	app.UsageText = "./ping-cli [global options] [hostname / IP]"
	app.HideHelpCommand = true
	
	app.Action = commandLineInterface
	app.Flags = []cli.Flag {
		&cli.StringFlag {
			Name: "ttl",
			Aliases: []string{"t"},
			Usage: "time to live",
			Value: "infinity",
		},
		&cli.StringFlag {
			Name: "count",
			Aliases: []string{"c"},
			Usage: "number of pings",
			Value: "infinity",
		},
		&cli.IntFlag {
			Name: "packetsize",
			Aliases: []string{"s"},
			Usage: "number of bytes to be sent",
			Value: defaultBytes,
		},
	}

	/* Run the application with provided arguments */
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

/* The main method running in the application */
func commandLineInterface(ctx *cli.Context) error {
	/* Check arguments */
	if argc := ctx.Args().Len(); argc != 1 {
		cli.ShowAppHelp(ctx)
		fmt.Println()
		return errors.New("Invalid number of arguments")
	}
	
	/* Parse input hostname / ip address as an ipv6 or ipv4 address*/
	addr, v4, err := parseIP(ctx.Args().Get(0))
	if err != nil {
		cli.ShowAppHelp(ctx)
		return err
	}

	/* Check the flag values and set the required variables */
	ttl := 0
	count := 0
	bytes := 56
	if ttlFlag := ctx.Int("ttl"); 0 < ttlFlag && ttlFlag < 256 {
		ttl = ttlFlag
	}
	if countFlag := ctx.Int("count"); 0 < countFlag {
		count = countFlag
	}
	if bytesFlag := ctx.Int("packetsize"); 0 < bytesFlag {
		bytes = bytesFlag
	}

	/* Send ICMP echo requests continuously to the address */
	fmt.Println("Pinging", addr, "with", bytes, "bytes of data:")
	i := 0
	successfulPings := 0
	totalPings := 0
	netRTT := time.Duration(0)
	maxRTT := time.Duration(0)
	minRTT := time.Duration(0)
	for count == 0 || i < count {
		/* Send the ICMP request and store whether or not a 
		   reply was recieved, before the request timed out */
		success, RTT, err := ping(addr, v4, ttl, bytes)
		if success {
			successfulPings++
			netRTT += RTT
			if RTT < minRTT || minRTT == 0 {
				minRTT = RTT
			}
			if RTT > maxRTT {
				maxRTT = RTT
			}
		} else if err != nil {
			fmt.Println(err)
		}
		totalPings++

		/* Output net packet loss as a percentage*/
		packetLoss := 100 - (100 * float64(successfulPings) / float64(totalPings))
		fmt.Printf("Packet Loss: %.2f%%", packetLoss)
		fmt.Println()

		/* Wait for a second before trying again */
		time.Sleep(time.Second)
		i++
	}

	/* Print statistics */
	packetLoss := 100 - (100 * float64(successfulPings) / float64(totalPings))
	fmt.Printf("Ping statistics for %v:\n", addr);
	fmt.Printf("\tPackets: Sent = %v, Received = %v, Lost = %v (%.2f%% loss),\n",
				totalPings, successfulPings, totalPings - successfulPings, packetLoss)
	fmt.Printf("Approximate round trip times in milli-seconds:\n");
	fmt.Printf("\tMinimum = %dms, Maximum = %dms, Average = %dms\n",
				 minRTT / time.Millisecond, maxRTT / time.Millisecond,
				netRTT / (time.Duration(successfulPings) * time.Millisecond))
	return nil
}

/* Creates a new connection and attempts to send a single ICMP echo request
   to the IP address addr along that connection, v4 indicates whether
   or not addr is an ipv4 or ipv6 address, ttl is the maximum number
   of hops the request is allowed to make before timing out, bytes is
   the number of bytes of data to send in the echo request */
// Inspired by: https://golang.hotexamples.com/examples/golang.org.x.net.icmp/Message/Marshal/golang-message-marshal-method-examples.html
func ping(addr *net.IPAddr, v4 bool, ttl int, bytes int) (bool, time.Duration, error) {
	var network string
	var listenAddr string
	var messageType icmp.Type
	var protocolNumber int
	
	/* Set variables unique to ipv4, ipv6 */
	if v4 {
		network = "ip4:icmp"
		listenAddr = "0.0.0.0"
		messageType = ipv4.ICMPTypeEcho
		protocolNumber = ipv4ProtocolNum
	} else {
		network = "ip6:ipv6-icmp"
		listenAddr = "::"
		messageType = ipv6.ICMPTypeEchoRequest
		protocolNumber = ipv6ProtocolNum
	}
	
	/* Open a connection and start listening for packets */
	c, err := icmp.ListenPacket(network, listenAddr)
    if err != nil {
		return false, 0, err
	}
	/* Close this connection once the function ends */
	defer c.Close()

	/* Set the time to live if the flag was set */
	if ttl != 0 {
		if v4 {
			c.IPv4PacketConn().SetTTL(ttl)
		} else {
			c.IPv6PacketConn().SetHopLimit(ttl)
		}
	}
	
	/* Generate an n byte message (n = bytes) */
	message := icmp.Message{
		Type: messageType, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: make([]byte, bytes),
		},
	}
	
	/* Produce the binary encoding of the message
	   including any added headers (8 bytes) */
	bin, err := message.Marshal(nil)
	if err != nil {
		return false, 0, err
	}

	/* Send the ICMP message */
	startTime := time.Now()
	_, err = c.WriteTo(bin, addr)
	if err != nil {
		return false, 0, err
	}

	/* Wait for a reply */
	replyBuffer := make([]byte, 1500)
	err = c.SetReadDeadline(time.Now().Add(readDeadline * time.Second))
	if err != nil {
		return false, 0, err
	}

	/* Read the reply */
	n, peer, err := c.ReadFrom(replyBuffer)
	if err != nil {
		return false, 0, err
	}

	/* Compute the round trip time */
	RTT := time.Since(startTime)

	/* Parses the reply into a message */
	repliedMessage, err := icmp.ParseMessage(protocolNumber, replyBuffer[:n])
    if err != nil {
        return false, 0, err
	}
	
	/* Checks the type of the recieved message, if the type is
	   a valid ipv4 or ipv6 ICMP echo reply the ping was a
	   success and we return true */
	switch repliedMessage.Type {
	case ipv4.ICMPTypeEchoReply:
		fallthrough
	case ipv6.ICMPTypeEchoReply:
		fmt.Printf("Reply from %v: bytes=%v time=%dms", peer, n, RTT / time.Millisecond)
		fmt.Println()
		return true, RTT, nil
	}
	return false, 0, nil
}

/* Parse string addr as an IP address, returns a pointer to the
   resolved ipv4 or ipv6 address, a boolean indicating whether
   it is ipv4 or ipv6, an error tracker */
func parseIP(addr string) (*net.IPAddr, bool, error) {
	var ip *net.IPAddr
	var err error
	parsedAddr := net.ParseIP(addr)
	v4 := false
	if parsedAddr == nil { /* Hostname */
		/* Try resolving as ipv6 address */
		ip, err = net.ResolveIPAddr("ip6", addr)
		if err != nil {
			/* Try resolving as ipv4 address */
			ip, err = net.ResolveIPAddr("ip4", addr)
			if (err != nil) {
				return nil, false, err
			}
			v4 = true
		}
	} else { /* ipv4 or ipv6 address */
		if ip4 := parsedAddr.To4(); ip4 != nil { /* ipv4 address */
			v4 = true
			ip, err = net.ResolveIPAddr("ip4", addr)
			if (err != nil) {
				return nil, false, err
			}
		} else { /* ipv6 address */
			ip, err = net.ResolveIPAddr("ip6", addr)
			if (err != nil) {
				return nil, false, err
			}
		}
	}
	return ip, v4, nil
}