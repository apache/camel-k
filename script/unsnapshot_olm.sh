#!/bin/sh

# Prefer unsnapshotting to regenerating, because changes done to snapshot file may get lost

location=$(dirname $0)
olm_catalog=${location}/../deploy/olm-catalog


for d in $(find ${olm_catalog} -type d -name "*-SNAPSHOT*");
do
  mv ${d} ${d//-SNAPSHOT/}
done

for f in $(find ${olm_catalog} -type f -name "*-SNAPSHOT*");
do
  mv ${f} ${f//-SNAPSHOT/}
done

for f in $(find ${olm_catalog} -type f);
do
  sed -i 's/-SNAPSHOT//g' $f
done
