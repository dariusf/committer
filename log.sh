#!/usr/bin/env bash

jq -n '[inputs | {no: input_line_number, name: .["_name"], state: del(.["_name"])}]' < log.ndjson > log.json
cp log.json ~/refinement-mappings/trace-specs/2pc/committer/Success.json
