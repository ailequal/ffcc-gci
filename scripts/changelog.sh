#!/bin/sh
# Reads CHANGELOG.md, which follows https://keepachangelog.com/en/1.1.0/.
#
#   scripts/changelog.sh check            fail if the file is not well formed
#   scripts/changelog.sh latest           print the newest released version, if any
#   scripts/changelog.sh notes <version>  print the notes of a released version
#
# The release workflow uses "latest" to decide what to release and "notes"
# for the text of the GitHub release. Set CHANGELOG to read another file.
set -eu

file=${CHANGELOG:-CHANGELOG.md}

case "${1:-}" in
check)
  awk '
    /^## \[/ {
      sections++
      if (sections == 1) {
        if ($0 != "## [Unreleased]") {
          print "the first section must be \"## [Unreleased]\", got: " $0
          bad = 1
        }
        next
      }
      if ($0 !~ /^## \[[0-9]+\.[0-9]+\.[0-9]+\] - [0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]$/) {
        print "not a \"## [X.Y.Z] - YYYY-MM-DD\" heading: " $0
        bad = 1
      }
      if (seen[$2]++) {
        print "version listed twice: " $2
        bad = 1
      }
    }
    /^### / && $0 !~ /^### (Added|Changed|Deprecated|Removed|Fixed|Security)$/ {
      print "not a Keep a Changelog type of change: " $0
      bad = 1
    }
    END {
      if (sections == 0) {
        print "there is no \"## [Unreleased]\" section"
        bad = 1
      }
      exit bad
    }
  ' "$file" >&2
  ;;

latest)
  sed -n -E 's/^## \[([0-9]+\.[0-9]+\.[0-9]+)\] - [0-9]{4}-[0-9]{2}-[0-9]{2}$/\1/p' "$file" | head -n 1
  ;;

notes)
  version=${2:?usage: scripts/changelog.sh notes <version>}
  # The section of the version, without its heading and without leading or
  # trailing blank lines. It ends at the next section or at the link list.
  notes=$(awk -v v="$version" '
    index($0, "## [" v "] ") == 1 { on = 1; next }
    on && (/^## \[/ || /^\[[^]]+\]: /) { exit }
    on && /^[[:space:]]*$/ { if (seen) blank++; next }
    on { while (blank > 0) { print ""; blank-- } print; seen = 1 }
  ' "$file")
  if [ -z "$notes" ]; then
    echo "$file has no notes for version $version" >&2
    exit 1
  fi
  printf '%s\n' "$notes"
  ;;

*)
  echo "usage: scripts/changelog.sh check | latest | notes <version>" >&2
  exit 2
  ;;
esac
