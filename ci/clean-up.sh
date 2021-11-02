#!/bin/bash

# this command purge all the buildpack cache - every smoketest run!!

# set tracing env var to true for more verbosity
export CF_TRACE=true
export CF_TRACE=./trace.log

# Main function

cf curl -v -X DELETE /v2/blobstores/buildpack_cache
