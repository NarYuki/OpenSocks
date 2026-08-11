#!/bin/sh
set -eu

version="${1#v}"
changelog="${2:-CHANGELOG.md}"

awk -v heading="## $version" '
  $0 == heading { found = 1; next }
  found && /^## / { exit }
  found { print }
  END { if (!found) exit 1 }
' "$changelog"
