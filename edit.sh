#!/usr/bin/env bash

set -eu

vim $(find */ -type f -name '*.go')
