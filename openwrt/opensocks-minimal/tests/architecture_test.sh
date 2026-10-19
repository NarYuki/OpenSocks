#!/bin/sh
# Run with BusyBox ash and hexdump to match OpenWrt.
set -eu

root="$(CDPATH= cd -- "$(dirname "$0")/../../.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT INT TERM

tr -d '\r' < "$root/openwrt/opensocks-minimal/files/usr/lib/opensocks/launcher.sh" \
	| sed -n '/^detect_binary_suffix() {$/,/^}$/p' \
	| sed "s|/bin/busybox|$work/elf|g" > "$work/launcher.sh"

# Render the actual postinst heredoc to catch premature build-time expansion.
eval "$(
	printf 'cat <<EOF\n'
	tr -d '\r' < "$root/tools/build-release.sh" | sed -n '/^detect_suffix() {$/,/^}$/p'
	printf '\nEOF\n'
)" | sed "s|/bin/busybox|$work/elf|g" > "$work/postinst.sh"

uname() { printf '%s\n' "$machine"; }
od() { return 127; }

check() {
	machine="$1"
	expected="$3"
	if [ "$2" = missing ]; then
		rm -f "$work/elf"
	else
		printf '\177ELF\001%b' "$2" > "$work/elf"
	fi

	status=0
	actual="$("$detect")" || status="$?"
	if [ "$expected" = unsupported ]; then
		[ "$status" -ne 0 ] && [ -z "$actual" ] || {
			printf 'FAIL %s: %s byte %s was accepted\n' "$source" "$machine" "$2" >&2
			exit 1
		}
	else
		[ "$status" -eq 0 ] && [ "$actual" = "$expected" ] || {
			printf 'FAIL %s: %s byte %s: expected %s, got %s (exit %s)\n' \
				"$source" "$machine" "$2" "$expected" "$actual" "$status" >&2
			exit 1
		}
	fi
}

for source in launcher postinst; do
	. "$work/$source.sh"
	case "$source" in
		launcher) detect=detect_binary_suffix ;;
		postinst) detect=detect_suffix ;;
	esac
	check mips '\001' mipsle
	check mips '\002' mips
	check mips '\000' unsupported
	check mips '' unsupported
	check mips missing unsupported
	check mipsel '\001' mipsle
	check aarch64 '\001' arm64
	check x86_64 '\001' amd64
	check unknown '\001' unsupported
	printf 'PASS %s: architecture detection without od (9 cases)\n' "$source"
done
