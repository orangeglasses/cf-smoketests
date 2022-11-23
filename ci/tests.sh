#!/bin/bash

# apt-get update
# apt-get install curl -y
# apt-get install jq -y

json=$(curl $CHECK_URL)
echo $json

jq_output="$(echo $json | jq -er ".[]")"
if [ $? -ne 0 ]
then
  echo "jq could not parse smoketests result"
  exit 1
fi

echo "${jq_output}" | grep false
if [ $? -eq 1 ]
then
  echo "all tests passed"
  exit 0
fi

exit 1
