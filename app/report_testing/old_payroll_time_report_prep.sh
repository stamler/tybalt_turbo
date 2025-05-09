#!/bin/bash

if [ -z "$1" ]; then
  echo "Usage: $0 filename"
  exit 1
fi

sed -i '' 's/,,/,0,/g' "$1"
sed -i '' 's/,,/,0,/g' "$1"
sed -i '' 's/,"",/,,/g' "$1"
