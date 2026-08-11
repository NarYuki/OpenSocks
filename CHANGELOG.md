# Changelog

## 0.2.2

### Security

- Added request-body, header-size, connection-concurrency, and HTTP timeout limits to the control servers.
- Serialized shared Transocks API state and connection operations to prevent concurrent session and configuration corruption.
- Pinned GitHub Actions, the LuCI build source, and the China IPv4 route list to verified immutable revisions or SHA-256 hashes.
- Added response-size limits and persistent connection management to the mobile client.
- Updated `flutter_secure_storage` and raised the supported Android API floor required by the newer secure-storage backend.

### Performance

- Replaced repeated per-setting UCI process launches with one snapshot read and one commit per settings update.
- Attached a classic BPF filter to the DNS observer so unrelated LAN frames are discarded in the kernel.
- Added 30-minute expiry to learned domain and service IPs, plus bounded daemon-side caches.
- Reduced persistent traffic writes from every five minutes to every fifteen minutes while preserving one-second in-memory statistics.
- Stopped hidden mobile tabs from polling traffic and prevented overlapping polling requests.
- Limited LuCI log reads to the latest 128 KiB and added 512 KiB log rotation.

### Reliability

- Added HTTP client/server timeouts and bounded concurrent request handling.
- Fixed races in API authentication state, settings refresh, and server switching.
- Corrected JSON error response headers.
- Verified downloaded China route data before applying nftables rules.
- Fixed immutable `by-sha` opkg URLs on HTTP servers that reject symbolic links.

### Distribution

- Published signed full, minimal, and LuCI packages through GitHub Releases and `rel.n4t.su`.
- Verified `Packages`, `SHA256SUMS`, and their usign signatures before publication.

### Validation

- Passed Go unit tests, `go vet`, the race detector, MIPS soft-float cross-compilation, Flutter analysis, and Flutter tests.
- Validated the signed minimal package and smart-routing nftables configuration on the target OpenWrt router.
