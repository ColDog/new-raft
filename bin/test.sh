#!/usr/bin/env bash


name=${1}
shift
go test github.com/coldog/raft/${name} -v -race "$@"
