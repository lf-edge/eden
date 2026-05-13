# Onboarding tests

End-to-end tests that exercise the EVE onboarding code path (zedclient,
i.e. `pkg/pillar/cmd/client`). These tests run against a live EVE
booted via the standard eden setup and complement the Go unit tests in
the eve repo under `pkg/pillar/cmd/client/`.

## Tests

- **onboarding_status** — asserts that after `eden eve onboard` the
  device has a valid `/persist/status/uuid`, a kernel hostname that
  matches it, and a non-empty `/persist/status/hardwaremodel`.
  Exercises zedclient's post-loop publishing path.

## Running

```sh
make build
eden test ./tests/onboarding/ -v
```

Or directly with the escript runner:

```sh
eden.escript.test -testdata ./tests/onboarding/testdata/ \
    -test.run TestEdenScripts/onboarding_status -test.v
```
