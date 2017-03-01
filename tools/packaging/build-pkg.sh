#!/usr/bin/env bash
set -eo pipefail

[[ $(uname) == Darwin || -n $JENKINS_HOME ]] && docker_cmd='docker' || docker_cmd='sudo docker'

BUILD_DIR="build"
BUILDER_PATH="deb"
BUILDER_COMMON_FILE="$BUILDER_PATH/build-common.sh"
POST_INSTALL_SCRIPT="$BUILDER_PATH/post-install.sh"
DISTRIBUTION="${DISTRIBUTION:-all}"

image_name="peloton-build"

# make sure we did a glide install before running the build

if [[ $DISTRIBUTION == all ]] ; then
  DISTRIBUTION="trusty jessie"
  echo "Building debs for all supported distributions ($DISTRIBUTION); set \$DISTRIBUTION to override"
fi

# make sure GOPATH is setup if missing. this is necessary to support building on
# uber ubuild machines without a valid gopath setup
# TODO(gabe): remove me when we no longer need a functioning GO env to build the docker
# containers.
if [[ -z ${GOPATH+x} ]] ; then
  workspace="$(pwd -P)/workspace"
  rm -rf "${workspace}" || :
  goDirPath="${workspace}/src/$(make project-name)"
  mkdir -p "$(dirname "$goDirPath")"
  if [ ! -e "$goDirPath" ]; then
    ln -sfv "$(dirname $workspace)" "$goDirPath"
  elif [ ! -L "$goDirPath" ]; then
    echo >&2 "error: $goDirPath already exists but is unexpectedly not a symlink"
    exit 1
  fi
  export GOPATH="$workspace"
fi

# FIXME(gabe) this is a hack; ensure deps are up to date before performing package build
# Remove this when we can properly perform a `glide install` in a container
make install

for dist in $DISTRIBUTION ; do
  echo "Building debs for $dist"
  outputdir="debs/$dist"
  [[ -d $outputdir ]] || mkdir -p $outputdir
  $docker_cmd build -t "$image_name" -f Dockerfile.deb.$dist .
  $docker_cmd run --rm -v "$(pwd)/${outputdir}":/output -t "$image_name"
  echo -e "\n\nDebs built:"
  ls -la $outputdir
  echo ""
done