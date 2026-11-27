
0.7.4
=============
2026-11-27

* Made QR codes from --add-client --qr about half as wide by drawing 2x2 modules per terminal cell with Unicode quadrant characters; the terminal font must include these characters. (c8cf33c)
* Added support for symlinked interface configs as in wg-quick. --list and port checks include them, client changes update the target file and keep the link, --delete removes only the link, and broken links are skipped with a warning. Previously, client operations replaced the link with a regular file. (9502812)
* Fixed --add-client with a mimicry profile on a wg tunnel. It now fails with exit code 3 instead of writing I1-I5 into a WireGuard config that wg-quick cannot load. (bcbc4e6)
* Fixed --keenetic on dual-stack tunnels. An IPv6 client address now fails with exit code 3 instead of being silently dropped; pass an IPv4-only --ip for Keenetic routers. (7f2eb37)
* Fixed --keenetic for AmneziaWG 3.0 configs without header protection. All AmneziaWG 3.x tunnels are now rejected with exit code 3. (5d4e45a)
* Changed --udp-listener --json to print {"port_start":N,"port_end":M,"status":"listening"} instead of the ports field, which was not valid JSON for port ranges. --udp-listener now rejects --timeout and --count with exit code 2 because it never used them. Update scripts that parse this output or pass these flags. (fb40b07)
* DEB and RPM packages now follow distribution policy. They include license and third-party notices, compressed manual pages, package changelogs generated from these release notes, and identical package descriptions, and both formats pass Lintian and rpmlint. (a07270a)
* Release binaries, packages, and SHA256SUMS are now reproducible byte for byte and carry signed GitHub build provenance; verify downloads with gh attestation verify as shown in the README. (86ce205)
* Updated the Go toolchain to 1.27.4. (a7d7d8c)

0.7.3
=============
2026-11-20

* Added application and manual page localization for all README languages. (fb2fe73)
* Expanded test coverage and CI quality gates, including UDP integration tests. (cb28408)
* Fixed tunnel deletion confirmation when the client set changes before confirmation. (1b31e6a)

