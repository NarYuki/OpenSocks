#!/bin/sh
set -u

dir=/tmp/opensocks
program="$dir/opensocks"
archive="$dir/opensocks.gz"
feed="https://rel.n4t.su/opkg"
feed_key="/etc/opkg/keys/d24a5e234001294c"

detect_binary_suffix() {
	case "$(uname -m)" in
		mipsel) printf '%s\n' mipsle ;;
		mips)
			# Some little-endian OpenWrt kernels report only "mips". Read the
			# ELF EI_DATA byte instead of guessing the byte order from uname.
			elf_data="$(od -An -j 5 -N 1 -tu1 /bin/busybox 2>/dev/null | tr -d ' ')"
			case "$elf_data" in
				1) printf '%s\n' mipsle ;;
				2) printf '%s\n' mips ;;
				*) return 1 ;;
			esac
			;;
		mips64el) printf '%s\n' mips64le ;;
		mips64) printf '%s\n' mips64 ;;
		aarch64|arm64) printf '%s\n' arm64 ;;
		armv7l|arm) printf '%s\n' arm ;;
		armv6l) printf '%s\n' arm6 ;;
		x86_64|amd64) printf '%s\n' amd64 ;;
		i386|i486|i586|i686) printf '%s\n' 386 ;;
		*) return 1 ;;
	esac
}

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
	suffix="$(detect_binary_suffix)" || return 1
	rm -f "$dir/Packages.new" "$dir/Packages.sig.new"
	wget -q -O "$dir/Packages.new" "$feed/Packages" || {
		rm -f "$dir/Packages.new" "$dir/Packages.sig.new"
		return 1
	}
	wget -q -O "$dir/Packages.sig.new" "$feed/Packages.sig" || {
		rm -f "$dir/Packages.new" "$dir/Packages.sig.new"
		return 1
	}
	usign -V -m "$dir/Packages.new" -p "$feed_key" -x "$dir/Packages.sig.new" >/dev/null 2>&1 || {
		rm -f "$dir/Packages.new" "$dir/Packages.sig.new"
		return 1
	}
	mv "$dir/Packages.new" "$dir/Packages"
	mv "$dir/Packages.sig.new" "$dir/Packages.sig"

	# Prefer architecture-specific metadata. Only mipsle may use the legacy
	# fields because releases before multi-architecture support were mipsle-only.
	new_sha="$(package_field "X-OpenSocks-Binary-SHA256-${suffix}")"
	new_url="$(package_field "X-OpenSocks-Binary-URL-${suffix}")"
	if { [ ${#new_sha} -ne 64 ] || [ -z "$new_url" ]; } && [ "$suffix" = "mipsle" ]; then
		new_sha="$(package_field X-OpenSocks-Binary-SHA256)"
		new_url="$(package_field X-OpenSocks-Binary-URL)"
	fi
	[ ${#new_sha} -eq 64 ] || return 1
	case "$new_sha" in *[!0-9a-f]*) return 1;; esac
	expected_name="opensocks-linux-${suffix}.gz"
	if [ "$new_url" != "$feed/by-sha/$new_sha/$expected_name" ]; then
		return 1
	fi
	uci set opensocks.settings.binary_sha256="$new_sha"
	uci set opensocks.settings.binary_url="$new_url"
	uci commit opensocks
}

download_and_verify() {
	suffix="$1"
	url="$(uci -q get opensocks.settings.binary_url)"
	sha="$(uci -q get opensocks.settings.binary_sha256)"
	[ -n "$url" ] && [ ${#sha} -eq 64 ] || return 1
	case "$sha" in *[!0-9a-f]*) return 1;; esac
	[ "$url" = "$feed/by-sha/$sha/opensocks-linux-${suffix}.gz" ] || return 1
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
binary_suffix="$(detect_binary_suffix)" || {
	echo "OpenSocks does not support CPU architecture: $(uname -m)" >&2
	exit 1
}
refresh_release_metadata || true
configured_sha="$(uci -q get opensocks.settings.binary_sha256)"
configured_url="$(uci -q get opensocks.settings.binary_url)"
[ "$configured_url" = "$feed/by-sha/$configured_sha/opensocks-linux-${binary_suffix}.gz" ] || {
	echo "OpenSocks has no verified binary for CPU architecture: $(uname -m)" >&2
	exit 1
}
if [ ! -s "$archive" ] || [ "$(sha256sum "$archive" 2>/dev/null | awk '{print $1}')" != "$configured_sha" ]; then
	rm -f "$archive" "$program" "$program.new"
fi
while [ ! -x "$program" ]; do
	download_and_verify "$binary_suffix" && break
	echo "OpenSocks download or verification failed; retrying in 30 seconds" >&2
	sleep 30
done
exec "$program" --daemon
