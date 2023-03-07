#!/bin/bash

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

docker build --rm -t log4cxx-tester-builder $DIR

# Work-around SELinux issue (https://github.com/docker/docker/pull/5910)
if sestatus; then
    echo "Setting SELinux label on project directory $DIR"
    chcon -Rt svirt_sandbox_file_t "$DIR"
fi

docker run --rm -v $DIR:/build log4cxx-tester-builder
docker rmi -f log4cxx-tester-builder
