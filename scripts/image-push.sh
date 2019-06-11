#!/bin/bash -x

echo "$QUAY_PASSWORD" | docker login -u "$QUAY_USERNAME" --password-stdin

make push
