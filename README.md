# CNI Benchmark Operator

[![Coverage Status](https://coveralls.io/repos/github/cni-benchmark/operator/badge.svg?branch=main)](https://coveralls.io/github/cni-benchmark/operator?branch=main)

Works in server or client modes.

## Server mode

Runs iperf3 server and waiting for connections infinitely.

## Client mode

Connects to iperf3 server, performs benchmark, analyzes JSON output and pushes data to the Database. Exits at the end.
