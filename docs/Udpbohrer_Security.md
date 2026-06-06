# Security Concept
## Assumptions
- Udpbohrer forwards UDP datagrams from Outside to Inside (behind NAT). This datagram exchange is always initiated from the Outside.
- Confidentiality and Integrity of the forwarded UDP datagrams is not in the scope of Udpbohrer. This has to be assured by the forwarded protocols like Wireguard or Openvpn.
- Udpbohrer secures its own control datagrams against tampering. However this does not include the source addresses of control datagrams: They may be modified by NAT routers (intended!) and malicious attackers.
- Malicious attackers with access to routers may insert, read or reroute packets. This must be handled by the forwarded protocols.
- The overhead of data packets is minimized. 
- As the forwarded protocol must handle Confidentiality and Integrity, the data of forwarded datagrams is neither encrypted nor checksummed. 
## HELO datagrams
HELO datagrams are periodically exchanged between Outside and Inside udpbohrer. They keep the UDP routing between both open despite the possible presence of NAT routers, track timeouts and communicate the address of the Inside proxy to the Outside proxy.

Their payload size is 10 bytes, the 6 byte extension is not used. Their content is:
- 4 byte timestamp
- 4 byte MAC
- 1 byte Id (always 0)
- 1 byte Flow (always 0)

The MAC is computed from Timestamp, Id and Flow. The timestamp is the lowest 32bit of the system time in seconds, which must be the same (with a defined tolerance) on Inside and Outside proxy. A wrap around every around 136 years is accepted, the exchange of HELO datagrams may fail during a couple of seconds around wrap around.

The HELO datagrams are wrapped into UDP packets and always originate from the Inside udpbohrer which sends them at defined intervals. Outside accepts them from all IPv4 or IPv6 addresses. On reception at the Outside, the timestamp is checked and the MAC is calculated again. If it matches, Outside stores the source address and port from the HELO packet,
recomputes the MAC and sends the HELO packet back. Inside receives the HELO packet and resets its timeout.

The source address of the HELO packet it *not* part of the MAC calculation. It may be changed by NAT routers beween Inside and Outside proxy.

# Data datagrams
The Outside proxy receives UDP packets from remote systems. A two byte header (Id and Flow) is generated and prepended to the data payload of the UDP datagram. This payload is wrapped in a new UDP datagram and forwarded through the funnel to the Inside proxy. The Inside proxy forwards it to the Local system.

- Each port on the Outside proxy is identified by one Id.
- The tuple (Remote source address, remote UDP source port, local host, local UDP port) is identified by the tuple (Id, Flow). Both are the header, Flow is allocated at random.

The Inside udpbohrer accepts data datagrams only from the Outside host and port defined in the configuration file. The Outside udpbohrer accepts data datagrams only from the source address and port from the last HELO datagram.

During transmission Id, Flow or Payload may be read or altered. Such events must be detected and handled by the forwarded protocol (usually VPN). 

## Cryptography

### Key Derivation
The cryptographic keys are calculated from entire configuration file and VERSION_MAJOR of the udpbohrer program. 

SHA512 hash = SHA52Hash(ConfigurationFile||VERSION_MAJOR)

The hash is split into:
- 128 bit Initialization Vector (IV) Inside to Outside
- 128 bit AES128 key Inside to Outside
- 128 bit Initialization Vector Outside to Inside
- 128 bit AES128 key Outside to Inside

Always use the same configuration file and the same program version on Outside and Inside. Otherwise the keys will not match.

### MAC Calculation
For MAC calculation the HELO datagram is padded to 16 byte and the MAC bytes are set to 0. Then the set of key and IV is used for encryption of the 16 byte datagram. The first 4 byte of the encrypted datagram are the MAC and copied into the original datagram.

On reception the MAC is stored and set to 0 in the received datagram. The MAC is calculated again and compared to the MAC from the received datagram. If the values do not match, the datagram is ignored and an error datagram is logged.

As the MAC calculation includes a timestamp for protection against replay attacks, the system time on both systems must be monotonous and within defined limits.

