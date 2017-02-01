#!/usr/bin/env bash

protoc -I=rpb/ rpb/raft.proto --go_out=plugins=grpc:rpb
