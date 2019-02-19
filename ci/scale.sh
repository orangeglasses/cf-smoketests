#!/bin/bash

# timeout is iterations times duration in seconds, so 12 * 10 = 120 seconds is 2 minutes.

iterations="12"
duration="10"

# login CloudFoundry

cf login -a $api -u $username -p $password -o $organization -s $space

# install autoscaler plugin

cf install-plugin pcf-app-autoscaler/autoscaler-for-pcf-cliplugin-linux64-binary-* -f

# the app to scale

appName="smokeTests"
appGUID=$(cf app $appName --guid)
echo 'Application Name: '$appName
echo 'Application GUID: '$appGUID

# configure autoscaling

cf configure-autoscaling $appName resource-git-smoketests/ci/autoscaler.yml
cf autoscaling-apps
cf autoscaling-rules $appName

# manually scale to 2 instances

echo "Scaling $appName to 2 instances."
cf scale $appName -i 2

# wait for autoscaler to act

echo "Waiting for autoscaler to scale $appName to 1 instance."

while [[ $iterations -gt 0 ]]; do
  instances=$(cf curl "/v2/apps/$appGUID" | jq .entity.instances)
  echo "Iteration: $iterations instances: $instances"
  if [[ "$instances" -eq 1 ]]; then
    echo "App is scaled down, exiting."
    cf autoscaling-events $appName | head -n 5
    exit 0
  fi
  sleep $duration
  iterations=$[$iterations-1]
done
echo "----------> Timed-Out!"
exit 1
