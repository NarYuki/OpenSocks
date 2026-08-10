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
assets="Packages Packages.gz Packages.sig SHA256SUMS SHA256SUMS.sig d24a5e234001294c luci-app-opensocks_${version}-1_all.ipk opensocks-linux-mipsle.gz opensocks-minimal_${version}-1_all.ipk opensocks_${version}-1_mipsel_24kc.ipk"

for asset in $assets; do
	curl -fL --retry 3 --connect-timeout 15 -o "$stage/$asset" "$base/$asset"
done

test "$("$usign_bin" -F -p "$stage/$public_key_fingerprint")" = "$public_key_fingerprint"
"$usign_bin" -V -m "$stage/Packages" -p "$stage/$public_key_fingerprint" -x "$stage/Packages.sig"
"$usign_bin" -V -m "$stage/SHA256SUMS" -p "$stage/$public_key_fingerprint" -x "$stage/SHA256SUMS.sig"
(cd "$stage" && sha256sum -c SHA256SUMS)
gzip -t "$stage/Packages.gz"
chmod 0755 "$stage"
chmod 0644 "$stage"/*

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
