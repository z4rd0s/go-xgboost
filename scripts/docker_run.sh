#!/bin/bash

set -e

docker run --rm -it -v "$PWD":/go/src/github.com/z4rd0s/go-xgboost -w /go/src/github.com/z4rd0s/go-xgboost xgboost-testing:latest $@