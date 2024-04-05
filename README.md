# Executor Repro

This sends a set of known witness + stream payloads to the list of executors and records the responses, comparing the stateroots with those calculated by cdk-erigon.

## To Run...

`go run ./...`

## Understanding output

- results output to 'output' dir in the repo root
- the result file will be named output_{batchno}_{serverip}.json
- the terminal will show unexpected stateroots