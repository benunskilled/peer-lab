# Security

## Reporting a vulnerability

If you find a security issue in Peer Lab, please report it privately through **Security → Report a vulnerability** in this repository.

Include the affected version, steps to reproduce the issue and its potential impact. Please leave out passwords, RPC credentials and other private information.

## Supported versions

Security fixes are provided for the latest release. Please update before checking whether an issue still occurs.

## Access to your node

Peer Lab uses Bitcoin Core's RPC and ZMQ interfaces. It reads peer and block information, Core's address book and block events, and uses `addnode` and `disconnectnode` to manage connections. While Stratum Race is on and a pool app on the same Umbrel takes part, it also asks Core for a block template after each new block, to time how long building it takes.

The app does not access wallet files, private keys or Core's block files, and does not edit `bitcoin.conf`. Slot Fill and Stratum Race are off on a fresh install.

Locations and network operators are looked up in databases bundled with the app, so no peer address is sent to an outside service. Your node's own place is looked up the same way.

On Umbrel, the app proxy handles dashboard authentication. The dashboard has no login of its own, so direct access to its internal port also gives access to its peer-management controls.

The dashboard only accepts changes from its own page. Every change, such as a reset, a disconnect or a switch, must arrive as a JSON request from the dashboard itself, so another website you have open cannot trigger it through your Umbrel login.

## Connections to mining pools

When Stratum Race is on, Peer Lab opens connections to the selected pools and reads their mining-job messages. It authorises with a public burn address and never submits mining shares.

While Stratum Race is off, no pool connections are open.

## Other reports

For ordinary bugs, display issues or wrong locations, please open a GitHub issue. Report problems in Bitcoin Core, Umbrel or pool software to their respective maintainers.
