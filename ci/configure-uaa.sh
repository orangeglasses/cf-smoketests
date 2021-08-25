#!/bin/sh

# Configure UAA before smoke tests
if $insecure; then
uaac target $uaatarget --skip-ssl-validation
else
uaac target $uaatarget
fi

uaac token client get $uaaadminuser -s "$uaaadminpassword"

# Set authorized grant types.
uaac client update $uaaclientid --authorized_grant_types "authorization_code,refresh_token,client_credentials,password" 

# Set authorities for the UAA client (it should be allowed to create a temporary user).
uaac client update $uaaclientid --authorities "scim.write,scim.read,uaa.resource"

# Set allowed scopes for UAA client (user should have smoketest.extinguish scope).
uaac client update $uaaclientid --scope "uaa.user,smoketest.extinguish"

