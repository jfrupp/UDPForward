# HOWTO use Tcpbohrer

## Definitions
- *Remote* systems (clients) initiate TCP connections to tcpbohrer Outside systems.
- The tcpbohrer *Outside* system acts as servers to remote systems and to the Inside system. It never initiates TCP connections, it accepts them.
- The tcpbohrer *Inside* system runs behind a NAT or a firewall. It initiates TCP control and data connections with the Outside system and with the Local systems. It never accepts connections.
- The *Local* system are severs behind a NAT or a firewall. Though tcpbohrer Inside and Outside they accept connections and exchange data with remote systems. 

## Operation
### Control connection
The control connection is established from the Inside to the Outside (OutHost, OutPort, tcp4 or tcp6). The Inside system sends periodic HELO messages to the Outside, the Outside replies with a HELO message. When the Outside system accepts a connection from Remote, it sends a control message over the control connection. As a TCP connection is supposed to tbe reliable, loss of control connection, HELO timeout or a badly formatted message are errors which cannot be recovered. Inside and Outside systems terminate and have to be restarted.

### Data connections
The establishment of data connections follows a sequence:
1. The Remote system initiates TCP connection A to the Outside system.
2. The Outside system accepts TCP connection A.
3. The Outside system sends a control message over the Control connection to the Inside system.
4. The Inside system checks and processes the control message and computes an answer to the he Control message.
5. The Inside system initiates TCP connection B to the Local system.
6. The Local system accepts TCP connection B.
7. The Inside system initiates TCP connection C to the Outside system, host and prot are the same as for Control connection.
8. The Outside system accepts TCP connection C.
9. The Inside system sends the answer to the control message over TCP connection C.
10. The Outside system uses the answer to the control message to identify the pair of TCP connections A and C.
11. The Outside system splits connections A and C into half connections for each directions and forwards data between them.
12. The Inside system splits connections B and C into half connections for each directions and forwards data between them.
13. The data exchange runs independently for each half connections until timeout or close of connection.

Out of Band data is not forwarded. The integrity of data and flow on data connections is handled by standard TCP mechanisms.
Encryption and further integrity has to be handled by the forwarded protocol (like VPN, SSH or TLS). Each tuple of TCP connections (A,B,C) has its own flow control. 
## Usage
Inside and Outside instances of tcpbohrer run with *identical* configuration files. Preferably they are compiled from the same source files, but binary files with the same major version are compatible as well. As cryptographic keys are calculated from major version and configuration file, a difference here leads to different keys. In this case communications between Inside and Outside fails. Check the *Config ID* that is printed during startup. It must be the same on Inside and Outside.

A sample configuration file is provided. It contains example parameters and comments. If the parameters are not described as  *optional*, they must be defined.

On the Inside, start with *tcpbohrer Inside tcpbohrer.yaml*. On the Outside, start with *tcpbohrer Outside tcpbohrer.yaml*.
Time between both systems must be roughly synchronized. If the Outside system is not running, the Inside system terminates immediately. Use a small delay between restarts.

Both processes will not detach from the terminal. In case of error they exit. Use the provided configuration files to start tcpbohrer from systemd and enforce restart in case of exit. 

### Port knock
Port knocking is optional and is configured globally in the `portknock` section, with an optional `portknockname` on each flow. The `portknock` block defines a TLS/HTTPS listener on a dedicated port, for example:

- `portknockport`: the HTTPS port used for port knocking
- `portknockkey`: private key for TLS
- `portknockcert`: certificate for TLS

A flow becomes reachable only after the remote client accesses the matching URL from the same source IP address. The pattern is:

`https://<outside-host>:<portknockport>/<portknockname>`

Example:

`https://outside.example.com:8888/1234`

The HTTP path must match the flow's `portknockname` exactly. Several flows may share the same knock name; in that case a successful knock from the same source address unlocks all matching flows. The port-knock state expires automatically after the configured timeout and must be refreshed by repeatedly loading the URL to keep the knock alive. Once a TCP data connection is already established, the port-knock is no longer required for that connection as long as it remains active.

The port-knock listener is started automatically by tcpbohrer when the `portknock` section is configured and at least one port knock name is configured in a flow. Without a port knock name in a flow, the TLS
port knock server will not start. 

Use a port knock TLS key and certificate only on the outside system and do not share it with other services. This protects the integrity of the
actual services which use separate certificates.  

IPV4 and IPV6 addresses must be treated separately. A port knock from an IPV4 address does not open a port for an IPV6 source address.

When using a web browser to trigger port knocking, keep the window or tab open for automatic refresh. The port will close when no further refreshes happen. When using tools like **curl** make sure that periodic refreshes happens.

**Set static routes between router, the system running tcpbohrer and the local servers!**

### TLS key generation for Port knock
``
openssl req -new \
            -x509 \
            -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 \
            -sha256 \
            -days 3650 \
            -noenc \
            -out ssl-cert-snakeoil.crt \
            -keyout ssl-cert-snakeoil.key
```
Only use this key/certificate key pair for port knock. Tell users
of the remote system to accept it for port knocking on the outside system.