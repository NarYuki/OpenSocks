#!/bin/sh
set -u

dir=/tmp/opensocks
program="$dir/opensocks"
archive="$dir/opensocks.gz"

download_and_verify() {
	url="$(uci -q get opensocks.settings.binary_url)"
	sha="$(uci -q get opensocks.settings.binary_sha256)"
	[ -n "$url" ] && [ ${#sha} -eq 64 ] || return 1
	case "$sha" in *[!0-9a-f]*) return 1;; esac
	rm -f "$archive.part" "$program.new"
	wget -q -O "$archive.part" "$url" || return 1
	[ "$(sha256sum "$archive.part" | awk '{print $1}')" = "$sha" ] || return 1
	gzip -t "$archive.part" || return 1
	mv "$archive.part" "$archive"
	gzip -dc "$archive" > "$program.new" || return 1
	chmod 0755 "$program.new"
	mv "$program.new" "$program"
}

mkdir -p "$dir"
while [ ! -x "$program" ]; do
	download_and_verify && break
	echo "OpenSocks download or verification failed; retrying in 30 seconds" >&2
	sleep 30
done
exec "$program" --daemon
