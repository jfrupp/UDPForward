# HOWTO use Udpbohrer

## Definitions
- *Remote* systems initiate UDP "connections" by sending packets to servers. The servers response is received by the Remote system. These remote systems may be behind a Network Address Translation (NAT) router, so the source address of packets may change during transmission. Quite often remote systems are VPN clients like Wireguard or OpenVPN.
- *Local* systems receive UDP "connections" by receiving packets from Remote systems and sending answers to them. These local systems are isolated from the Internet due to firewalls or NAT and cannot be directly reached from remote systems. Usually local systems are VPN servers which serve a local network.
- The *Outside* systems runs Udpproxy in "Outside" configuration. It receives UDP packets from remote systems, funnels them to Inside system and forwards packets from Inside back to the Remote system.
- The *Inside* system runs Udpproxy in "Inside" configuration. It receives UDP packets from Outside systems through the Funnel, forwards them to local system and forwards the Local system's answer back through the funnel.
- The *Funnel* is the UDP "connection" between Outside and Remote systems. It carries HELO control messages which keep the funnel alive and through these messages the Outside system learns address and port of the Inside system. 
- The *Local* side is a UDP server running on the behind a NAT, often on the same system as udpbohrer. It cannot be reached directly from the Outside. Instead UDP messages are sent between Inside and Local.

The *Outside* system must have a public IP address (IPV4 and/or IPV6).  

## Usage
Inside and Outside instances of udpbohrer run with *identical* configuration files. Preferably they are compiled from the same source files, but binary files with the same major version are compatible as well. As cryptographic keys are calculated from major version and configuration file, a difference here leads to different keys. In this case communications between Inside and Outside fails. Check the*Config ID*that is printed during startup. It must be the same on Inside and Outside.

A sample configuration file is provided. It contains example parameters and comments. If the parameters are not described as  *optional*, they must be defined.

On the Inside, start with *udpbohrer Inside udpbohrer.yaml*. On the Outside, start with *udpbohrer Outside udpbohrer.yaml*.

Both processes will not detach from the terminal. In case of error they exit. Use the provided configuration files to start udpbohrer from systemd and enforce restart in case of exit. 

**Set static routes between router, the system running udpbohrer and the local servers!**