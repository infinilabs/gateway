 #!/bin/bash

#init
PNAME=gateway
STABLE=/infini/Sync/Release/$PNAME/stable

if [[ $VERSION =~ NIGHTLY ]]; then
  exit 1
fi

for t in 386 amd64 arm64 armv5 armv6 armv7 loong64 mips mips64 mips64le mipsle riscv64 ; do
  cp -rf ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-linux-$t.tar.gz $STABLE
done

for t in mac-amd64 mac-arm64 windows-amd64 windows-386 ; do
  cp -rf ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-$t.zip $STABLE
done

