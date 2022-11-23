#!/bin/sh

# set tracing env var to true for more verbosity - set by martin caarels oct 5 2017
export CF_TRACE=true
export CF_TRACE=./trace.log

appName="smokeTests"

if $insecure; then
cf login -a $api -u $username -p $password -o $organization -s $space --skip-ssl-validation
else
cf login -a $api -u $username -p $password -o $organization -s $space
fi

tempManifest="tempmanifest.yml"
cd ./resource-git-smoketests/ &&\
cat $manifest > $tempManifest &&\
cat $services >> $tempManifest &&\
cf push -f $tempManifest

# eval "cat ../trace.log; exit $?"


