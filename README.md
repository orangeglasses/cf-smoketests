# Smoketests
Application that performs smoketests and sends results to `$DASHBOARD_DATA_ENDPOINT`.

## Local testing
1. `cf ssh` to app on environment you want to test against.
2. Use `env` command to obtain `VCAP_SERVICES`, `ADFS_RES_URL`, `UAA_RES_URL`, `VCAP_APPLICATION` and export them
3. Export `PORT` variable
4. `go build && ./SmokeTests`
5. In separate terminal `curl localhost:$PORT/v1/status`

### How to set the smoketests pipeline

```
git clone git@gitlab.unix.int.politie:haas/cf-smoketests.git
cd cf-smoketests/ci
fly -t (foundation) sp -p smoketests -c pipeline.yml  -l creds.yml -v foundation=(foundation)

```
