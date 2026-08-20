#!/bin/sh
set -eu

repository="${GITHUB_REPOSITORY:-NarYuki/OpenSocks}"
publish_root="${PUBLISH_ROOT:-/srv/opensocks-repo}"
usign_bin="${USIGN:-/usr/local/bin/usign}"
public_key_fingerprint="d24a5e234001294c"
api="https://api.github.com/repos/$repository/releases/latest"

tag="${RELEASE_TAG:-}"
if [ -z "$tag" ]; then
	tag="$(curl -fsSL -H 'Accept: application/vnd.github+json' "$api" |
		jq -er '.tag_name')"
fi

case "$tag" in
	v[0-9]*) ;;
	*) printf 'Invalid release tag: %s\n' "$tag" >&2; exit 1 ;;
esac
version="${tag#v}"

command -v curl >/dev/null
command -v jq >/dev/null
test -x "$usign_bin"

stage="$(mktemp -d "$publish_root/.opkg-${tag}.XXXXXX")"
trap 'rm -rf "$stage"' EXIT INT TERM
base="https://github.com/$repository/releases/download/$tag"
cache_bust="$(date -u +%s)"
metadata_assets="Packages Packages.gz Packages.sig SHA256SUMS SHA256SUMS.sig d24a5e234001294c"

for asset in $metadata_assets; do
	curl -fL --retry 3 --connect-timeout 15 -o "$stage/$asset" "$base/$asset?cache=$cache_bust"
done

test "$("$usign_bin" -F -p "$stage/$public_key_fingerprint")" = "$public_key_fingerprint"
"$usign_bin" -V -m "$stage/Packages" -p "$stage/$public_key_fingerprint" -x "$stage/Packages.sig"

# Collect every Filename basenames referenced by the signed Packages index.
package_assets="$(
	awk '
		BEGIN { RS=""; FS="\n" }
		{
			for (i = 1; i <= NF; i++) {
				if ($i ~ /^Filename: by-sha\//) {
					sub(/^Filename: by-sha\/[0-9a-f]+\//, "", $i)
					print $i
				}
			}
		}
	' "$stage/Packages" | sort -u
)"

# Also publish every gzipped daemon binary listed in SHA256SUMS.
binary_assets="$(
	awk '
		{
			name=$2
			sub(/^\.\//, "", name)
			if (name ~ /^opensocks-linux-.*\.gz$/) print name
		}
	' "$stage/SHA256SUMS" | sort -u
)"

assets="$(printf '%s\n%s\n' "$package_assets" "$binary_assets" | awk 'NF && !seen[$0]++')"
[ -n "$assets" ] || {
	printf 'No package assets found in Packages/SHA256SUMS\n' >&2
	exit 1
}

printf '%s\n' "$assets" > "$stage/.asset-list"
while IFS= read -r asset; do
	[ -n "$asset" ] || continue
	case "$asset" in
		luci-app-opensocks_${version}-*_all.ipk) ;;
		opensocks-minimal_${version}-*_all.ipk) ;;
		opensocks_${version}-*_*.ipk) ;;
		opensocks-linux-*.gz) ;;
		*)
			printf 'Unexpected release asset from index: %s\n' "$asset" >&2
			exit 1
			;;
	esac
	curl -fL --retry 3 --connect-timeout 15 -o "$stage/$asset" "$base/$asset?cache=$cache_bust"
done < "$stage/.asset-list"
rm -f "$stage/.asset-list"

"$usign_bin" -V -m "$stage/SHA256SUMS" -p "$stage/$public_key_fingerprint" -x "$stage/SHA256SUMS.sig"
(cd "$stage" && sha256sum -c SHA256SUMS)
gzip -t "$stage/Packages.gz"

# Cloudflare may ignore query strings in its cache key. Publish immutable,
# content-addressed aliases so updated packages and boot binaries can never be
# confused with an older asset that used the same release filename.
while read -r checksum relative; do
	name="${relative#./}"
	case "$name" in
		Packages|Packages.gz|Packages.sig|SHA256SUMS|SHA256SUMS.sig|"$public_key_fingerprint") continue ;;
	esac
	mkdir -p "$stage/by-sha/$checksum"
	# Use a same-filesystem hard link. Some HTTP servers intentionally refuse
	# symlinks outside their configured path even when the resolved file stays
	# below the document root, which makes opkg's immutable URL return 404.
	ln "$stage/$name" "$stage/by-sha/$checksum/$name"
done < "$stage/SHA256SUMS"

find "$stage" -type d -exec chmod 0755 {} +
find "$stage" -type f -exec chmod 0644 {} +

generation=".opkg-$tag-$(date -u +%Y%m%dT%H%M%SZ)"
final="$publish_root/$generation"
mv "$stage" "$final"
trap - EXIT INT TERM

next="$publish_root/.opkg.next"
ln -s "$generation" "$next"
if [ -d "$publish_root/opkg" ] && [ ! -L "$publish_root/opkg" ]; then
	mv "$publish_root/opkg" "$publish_root/opkg.before-sync"
fi
mv -Tf "$next" "$publish_root/opkg"

find "$publish_root" -maxdepth 1 -type d -name '.opkg-v*' -mtime +7 -exec rm -rf {} +
printf 'Published %s from GitHub Release %s\n' "$final" "$tag"
