# Peer Lab

One Umbrel app for your node's peers: where they are and what they are, which
of them brings new blocks first, Slot Fill for Core's manual slots, and Stratum
Race for mining pools. Test builds for now, app ID `bltest-peerlab`, port
**8796**. Bitcoin Node is the only dependency.

It replaces Bitcoin Lab and Peer Map. Turn off Bitcoin Lab's Slot Fill, or
uninstall Bitcoin Lab and Peer Map, before you install Peer Lab. Peer Lab
keeps the manual peers already set in Core, but starts with its own data;
nothing is imported.

## Layout

- `app/` – the page, measurements, Slot Fill, Stratum Race (Node)
- `peers/` – the peer list, places and peer classification (Go), served under `/peers/`
- `Dockerfile` – one image for all five services

Settings: [app/docs/configuration.md](app/docs/configuration.md).

## Build and test

```sh
cd app && npm ci && PEERS_DIR=../peers npm test
cd ../peers && go test ./...
```

CI also runs both against real Bitcoin Core on regtest (`REQUIRE_BITCOIND=1 node test/integration/regtest.js`, `go test -tags integration .`) and loads the page in a browser.

A version tag builds the image for amd64 and arm64, runs it on both and
prints the reference with its digest that the store entry pins. Test builds
are published from
[bitcoin-lab-test-store](https://github.com/benunskilled/bitcoin-lab-test-store).

## Licence

MIT. Geolocation and ASN data by DB-IP (CC BY 4.0), map data by Natural Earth
(public domain). Sources of the bundled data: `peers/geo/SOURCE.md`,
`peers/asn/SOURCE.md`, `peers/torexit/SOURCE.md`, `app/src/data/`.
