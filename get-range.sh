#!/usr/bin/env bash
set -euo pipefail

DD=$1
MM=$2
YYYY=$3
for HH in {0..23}
do
    paddedHH=$(printf "%02d\n" $HH)
    paddedMM=$(printf "%02d\n" $MM)
    paddedDD=$(printf "%02d\n" $DD)
    datestring=$YYYY-$paddedMM-${paddedDD}T${paddedHH}:00:00
    echo "at ${datestring}"
    ./scraper download --hour $HH --day $DD --month $MM --year $YYYY --output ${datestring}.grb2
    ./scraper process --input ${datestring}.grb2 --output ${datestring}.jpeg
    rm ${datestring}.grb2
    mv ${datestring}.jpeg files/
done
