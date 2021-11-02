#!/bin/bash

# this command purge all the buildpack cache - every smoketest run!!

# Main function

cf login -a $api -u $username -p $password -o platformteam -s smoketest --skip-ssl-validation

cf curl -v -X DELETE /v2/blobstores/buildpack_cache
