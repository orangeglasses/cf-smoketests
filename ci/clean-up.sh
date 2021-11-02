#!/bin/bash

# this command purge all the buildpack cache - every smoketest run!!

# Main function

if $insecure; then
cf login -a $api -u $username -p $password -o $organization -s $space --skip-ssl-validation
else
cf login -a $api -u $username -p $password -o $organization -s $space
fi

cf curl -v -X DELETE /v2/blobstores/buildpack_cache
