#!/usr/bin/env bash

set -ex
outDir=output
PluginName=decoder
buildFlag="-X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.Version=$(git rev-parse --short HEAD)  -X hook.PluginName=${PluginName}"
go build -ldflags="${buildFlag}" -o ${outDir}/${PluginName}.exe  ./cmd/decode/main.go
go build -ldflags="${buildFlag}" -o ${outDir}/hls.exe ./cmd/hls/main.go
