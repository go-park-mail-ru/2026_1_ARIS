#!/bin/sh

set -eu

cd "$(dirname "$0")/.."

go run ./tools/seed-media/cmd
