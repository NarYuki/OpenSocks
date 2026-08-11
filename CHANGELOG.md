# Changelog

## 0.2.9

### Routing controls

- Fixed the mobile routing sheet dropping Smart/Full China and Single/Dual/Triple changes while restoring the native navigation bar.
- Kept the routing operation active after the sheet closes so authentication, network configuration, interface processing, completion, and errors remain visible on the main screen.
- Fixed the LuCI proxy omitting `session_count` from settings requests.
- Prevented LuCI status polling from replacing edited routing values while a settings request is running.
- Kept the selected routing and session modes after service and full-router restarts.

## 0.2.8

### Multi-session UDP routing

- Enabled UDP forwarding on every active session in Dual and Triple modes.
- Added symmetric flow hashing so each UDP flow remains pinned to one session while separate flows are distributed across the available sessions.
- Kept the same policy-routing mark on every UDP slot so all selected flows use the existing TPROXY route safely.

## 0.2.7

### Mobile interface

- Unified Smart/Full China routing and Single/Dual/Triple session selection in one routing-mode sheet.
- Displayed the selected routing mode and session mode together on the home routing card.
- Applied both selections in one network reconfiguration and removed the duplicate session selector from Settings.

## 0.2.6

### Multi-session routing

- Added Single, Dual, and Triple session modes as supported operating modes in LuCI and the mobile client.
- Kept Single mode as the default for new installations while preserving the selected mode across service and router restarts.
- Connected every active session to the selected Transocks line and distributed new China-bound TCP flows across two or three isolated routes with nftables.
- Added active-session reporting to the control API and mode-aware status indicators to LuCI and the mobile client.
- Preserved compatibility with installations that previously enabled the dual-session setting.

### Speed testing

- Automatically selected the speed-test stream count for each session mode.
- Added three-session speed-test pooling with six internal transfer streams in Triple mode.
- Improved the China-route benchmark connection pool, idle-connection cleanup, progress update frequency, and memory reclamation on low-memory routers.
- Expanded Ookla endpoint selection and improved failure handling for SpeedTest.cn and external mainland test servers.

### Traffic and routing

- Expanded China-service classification and corrected service names in the traffic ranking.
- Improved DNS observation and dynamic service-address tracking for subdomains and short-lived endpoints.
- Added balanced two-way and three-way nftables TCP dispatch while retaining the primary session for UDP routing.
- Paused background traffic sampling and route recovery during speed tests to reduce CPU and memory contention.

### Reliability

- Added isolated persistent state for each active session and made multi-session login and connection setup failure-safe.
- Improved connection retry behavior for transient Transocks authentication failures.

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
- Rejected invalid China route data before applying nftables rules.
- Fixed immutable `by-sha` opkg URLs on HTTP servers that reject symbolic links.

### Distribution

- Published signed full, minimal, and LuCI packages through GitHub Releases and `rel.n4t.su`.
