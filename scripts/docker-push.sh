#!/bin/bash -x
set -ev

if [ "${TRAVIS_PULL_REQUEST}" = "false" ] || [ "${FORCE_DOCKER_PUSH}" = "true" ]; then
    echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
    DOCKER_ID_USER="$DOCKER_USERNAME" make pushimage
fi
