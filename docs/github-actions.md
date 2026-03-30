# Github Actions

Eden is a part of testing infrastructure of EVE and it's integrated in EVE CI/CD pipelines. EVE uses [test.yml](https://github.com/lf-edge/eden/blob/master/.github/workflows/test.yml) reusable workflow to run eden tests against specific EVE version in PR.

## Using GitHub Cache to run `test.yml` with custom EVE build

Sometimes you want to run tests in your CI/CD with EVE version, which is not published on Dockerhub,
for instance, when you have pull request to master. Eden [will](https://github.com/lf-edge/eden/blob/ed507793968a2005212d589d6c3d88824783a9a7/pkg/utils/container.go#L175-L178) prefer local image over pulling from Dockerhub. That means if you load image before running tests it will work with local image. For workflow `test.yml` you can use `eve_image_cache_key` parameter

### Why GitHub Artifacts and `eve_artifact_name` parameter?

In order to pass objects between jobs you need to either use cache or artifacts. Artifacts are published and stored for 90 days, cache is not published and GitHub deletes previous entries if total amount of cache is more than 10GBs. Artifacts also rebuild each time.

Unfortunately, you can't add additional steps before invoking reusable workflow, otherwise we could have just do `docker load` before invoking tests workflow.

**Important note:** Archive you store in GitHub Artifacts should be `eve_artifact_name`.tar

## Docker Hub Pull-Through Registry Mirror

Self-hosted runners use a local registry mirror (pull-through cache) for Docker Hub image pulls. This avoids Docker Hub rate limits and speeds up image fetching.

The mirror is configured via the `EDEN_REGISTRY_MIRROR` environment variable, which should be set at the project or organization level in GitHub Actions. When set, the [setup-environment](../.github/actions/setup-environment/action.yml) action automatically configures Eden to use it:

```shell
./eden config set default --key=registry.mirror --value="$EDEN_REGISTRY_MIRROR"
```

This sets the `registry.mirror` config key, which Eden passes to EVE as the datastore FQDN for pulling application images (see `getDataStoreFQDN` in `pkg/expect/docker.go`). Instead of pulling directly from Docker Hub, EVE pulls from the local mirror, which caches images transparently. Because of this, authenticated Docker Hub logins are not needed for pulling images in test workflows.
