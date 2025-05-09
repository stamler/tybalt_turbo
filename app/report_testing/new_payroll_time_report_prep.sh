#!/bin/bash

if [ -z "$1" ]; then
  echo "Usage: $0 filename"
  exit 1
fi

perl -i -pe 's/(?<=^|,)(?!")((?!(?:TRUE|FALSE)(?:,|$))[^",\r\n]*[A-Za-z_][^",\r\n]*)(?=,|$)/"\1"/g' "$1"
sed -i '' 's/,TRUE/,true/g' "$1"
sed -i '' 's/,FALSE/,false/g' "$1"
