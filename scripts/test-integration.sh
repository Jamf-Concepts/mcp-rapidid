#!/bin/bash
# Copyright 2026, Jamf Software LLC

set -e

cd $(dirname "$0")/..
go test -tags integration ./test/integration/
