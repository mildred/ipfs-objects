#!/bin/sh

cd "$(dirname "$0")"

(
echo "#!/usr/bin/sed -rf"
cd src
for f in gx/ipfs/*/*; do
  pkg="${f##*/}"
  echo 's|"gx/ipfs/[^/]*/'"$pkg"'(/[^"]*)?"|"'"$f"'\1"|'
done
) | tee "${0%.sh}.sed"
chmod +x "${0%.sh}.sed"

find src -name *.go | grep -Ev 'gx/ipfs|golang.org|github.com' | while read f; do
  ( set -x;
    sed -i -rf "${0%.sh}.sed" "$f"
  )
done
