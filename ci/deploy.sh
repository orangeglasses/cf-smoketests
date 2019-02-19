#!/bin/sh
#set
#pwd 
# installing curl and jq
# apt-get update
# apt-get install curl -y


# set tracing env var to true for more verbosity - set by martin caarels oct 5 2017
export CF_TRACE=true
export CF_TRACE=./trace.log

appName="smokeTests"

cf login -a $api -u $username -p $password -o $organization -s $space &&\
cd ./resource-git-smoketests/ &&\
cf push -f $manifest

# eval "cat ../trace.log; exit $?"


