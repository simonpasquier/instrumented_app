#!/bin/bash -x

echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin

DOCKER_ID_USER="$DOCKER_USERNAME" make pushimage
