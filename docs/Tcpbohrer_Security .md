# Security Concept
## Assumptions
- Tcpbohrer forwards TCP connections from Outside to Inside (behind NAT or firewall). 
- Confidentiality and Integrity of the forwarded TCP connections in the scope of tcpbohrer. This has to be assured by
the forwarded protocols like TLS, SSH  or Openvpn.
- Tcpbohrer secures its own control messages against tampering. However this does not include the IP addresses. They may be modified by NAT routers (intended!) and malicious attackers.
- Malicious attackers with access to routers may insert data into connections, read or reroute connections. This must be handled by the forwarded protocols.
- As the forwarded protocol must handle Confidentiality and Integrity, the data of forwarded connections is neither encrypted nor checksummed. 
## HELO messages
HELO messages are periodically exchanged between Outside and Inside tcpbohrer. They assure mutual communications between Inside and Outside systems.

Their payload size is 14 bytes.
- 4 byte timestamp
- 4 byte MAC
- 1 byte Id (always 0)
- 1 byte Flow (always 0)
- 4 byte Padding

The MAC is computed from the rest of the message. The timestamp is the lowest 32 bits of the system time in seconds, which must be the same (with a defined tolerance) on Inside and Outside proxy. A wrap around every around 136 years is accepted, the exchange of HELO messages may fail during a couple of seconds around wrap around.

## CONNECTION messages
When the Outside system has accepted a connection, it sends a connection message. Its size is 14 bytes:
- 4 byte timestamp
- 4 byte MAC
- 1 byte Id (always 0)
- 1 byte Flow (always 0)
- 4 byte Random nonce

Before sending out this message, the Outside system encodes it twice. Once for sending to the Inside and a second time for parking the Outside connection in a dictionary/map. 

The sequence is as follows:
1. Outside system accepts a connection from a Remote system.
2. The Outside system computes a CONNECTION message for sending to Inside.
3. The Outside system computes a CONNECTION message (different MAC) for parking the connection.
4. The Outside sends the CONNECTION message to the Inside.
5. The inside system verifies the CONNECTION message.
6. If OK, the Inside system computes a CONNECTION message for sending to the Outside.
7. The Inside system establishes a new connection to Outside.
8. The Inside system sends back the CONNECTION message on the new connection to the Outside.
9. if everything is OK, data exchange starts.

Malformatted messages on the control connection trigger immediate exit of Inside and Outside. The timeouts / keep alives for control and data connections must keep the connections open despite the presence of NAT systems or firewalls.

## Cryptography
See security of udpbohrer.

