# Benchmarking setup

This repo contains the benchmarking setup for comparing lamu and django.


## Testing methodology

This benchmark has 3 workflows, and a configurable number of workers run one of those 3 workflows in a loop in random.
Postgres max connections is altered to be 10000
