#!/bin/sh
set -u

dir=/tmp/opensocks
program="$dir/opensocks"
archive="$dir/opensocks.gz"
feed="https://rel.n4t.su/opkg"
feed_key="/etc/opkg/keys/d24a5e234001294c"

package_field() {
	field="$1"
	awk -v wanted="$field: " '
		BEGIN { RS=""; FS="\n" }
		$0 ~ /(^|\n)Package: opensocks-minimal(\n|$)/ {
			for (i = 1; i <= NF; i++)
				if (index($i, wanted) == 1) {
					print substr($i, length(wanted) + 1)
					exit
				}
		}
	' "$dir/Packages"
}

refresh_release_metadata() {
	command -v usign >/dev/null 2>&1 || return 1
	[ -s "$feed_key" ] || return 1
	rm -f "$dir/Packages.new" "$dir/Packages.sig.new"
	wget -q -O "$dir/Packages.new" "$feed/Packages" || return 1
	wget -q -O "$dir/Packages.sig.new" "$feed/Packages.sig" || return 1
	usign -V -m "$dir/Packages.new" -p "$feed_key" -x "$dir/Packages.sig.new" >/dev/null 2>&1 || return 1
	mv "$dir/Packages.new" "$dir/Packages"
	mv "$dir/Packages.sig.new" "$dir/Packages.sig"
	new_sha="$(package_field X-OpenSocks-Binary-SHA256)"
	new_url="$(package_field X-OpenSocks-Binary-URL)"
	[ ${#new_sha} -eq 64 ] || return 1
	case "$new_sha" in *[!0-9a-f]*) return 1;; esac
	[ "$new_url" = "$feed/by-sha/$new_sha/opensocks-linux-mipsle.gz" ] || return 1
	uci set opensocks.settings.binary_sha256="$new_sha"
	uci set opensocks.settings.binary_url="$new_url"
	uci commit opensocks
}

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
refresh_release_metadata || true
configured_sha="$(uci -q get opensocks.settings.binary_sha256)"
if [ ! -s "$archive" ] || [ "$(sha256sum "$archive" 2>/dev/null | awk '{print $1}')" != "$configured_sha" ]; then
	rm -f "$archive" "$program" "$program.new"
fi
while [ ! -x "$program" ]; do
	download_and_verify && break
	echo "OpenSocks download or verification failed; retrying in 30 seconds" >&2
	sleep 30
done
exec "$program" --daemon
