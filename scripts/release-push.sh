#!/bin/bash -ex

make promu
promu crossbuild -v
promu crossbuild tarballs
promu checksum .tarballs
promu release .tarballs
