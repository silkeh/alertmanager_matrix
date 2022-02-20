#!/bin/bash -e

which rpmspec >/dev/null 2>&1 && {
    DIR=`rpmspec -q --queryformat '%{name}-%{version}\n' *spec | head -1`
} || {
    which git >/dev/null 2>&1 || {
        echo 'rpmspec is not available and git is not available.  Please install the rpm-build package with the command `dnf install rpm-build` to continue, then rerun this step.' && exit 1
    }
    ver=$(git describe --tags) || {
        echo 'git describe --tags failed to find a tag that matches the current commit' && exit 1
    }
    ver=$(echo "$ver" | sed 's/^v//')
    DIR=$(basename "$PWD")
    DIR="$DIR-$ver"
}
echo "$DIR"
