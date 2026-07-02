# Eden setup for the kvm-to-k boot-disk repartition test — shrink variant

`update_eve_image_kvm_to_k_resize.txt` drives the EVE-kvm → EVE-k in-field
boot-disk repartition (small → large GPT geometry) for the default field layout,
where **P3 fills the disk**, so the resizer must **shrink the ext4 `/persist`**
first to free room, then grow ESP/IMGA/IMGB. It is the shrink sibling of
`update_eve_image_kvm_to_k_grow.txt` (the grow-only variant, where the disk
already has a free tail and P3 is left unchanged); the two escripts diff only on
those variant specifics.

The repartition runs **offline in storage-init**: baseosmgr backs up the
device-identity files to the CONFIG partition, arms a shrink flag, and reboots;
storage-init shrinks `/persist` then grows the partitions while `/persist` is
unmounted (the boot disk's GPT can't be re-read live while its rootfs is mounted).
The test asserts:

1. the disk **started small** (P3 filling it — the `shrink` case);
2. ESP/IMGA/IMGB **grew to the large geometry** and **P3 shrank** to make room;
3. the repartition **preserved the TPM seal** — checked post-hoc from
   `/persist/newlog` (the last boot on the conversion-capable kvm image unsealed
   locally; no boot re-sealed due to PCR5). The EVE-k boot's controller-key unlock
   is the expected new-rootfs behavior, not a failure;
4. an app **redeploy reused cached blobs** (downloader `< 1 MiB`), thanks to
   `timer.defer.content.delete=24h` holding the blobs across the delete + convert.

## Image sequence (3 images — build these first)

| Step | Image | HV | Layout | Source |
|------|-------|----|--------|--------|
| bringup | `12.1.0` (or any small-layout release) | kvm | **small** (36 MiB ESP, 300/512 MiB IMGA/IMGB) | published `lfedge/eve` |
| 2 | `kvm-to-k-resize-<sha>` | kvm | small (unchanged) | local build |
| 3 | `kvm-to-k-resize-<sha>` | k | large (after repartition) | local build |

The kvm→kvm hop (step 2) lands the conversion code on the small layout without
changing geometry; the kvm→k hop (step 3) triggers the repartition. Build the two
local images from the kvm-to-k-resize eve workspace (must go through the build
script — see CLAUDE.md):

```sh
~/bin/eve-build.sh ~/lf-edge/work/kvm-to-k-resize/eve kvm   # -> lfedge/eve:0.0.0-kvm-to-k-resize-<sha>-kvm-amd64
~/bin/eve-build.sh ~/lf-edge/work/kvm-to-k-resize/eve k     # -> lfedge/eve:0.0.0-kvm-to-k-resize-<sha>-k-amd64
```

`RESIZE_EVE_VER` is the part **before** `-kvm-amd64` / `-k-amd64`, e.g.
`0.0.0-kvm-to-k-resize-<sha>`.

## eden bringup (TPM required)

The vault assertions need a TPM, so eden must run with `eve.tpm=true` + swtpm.
Bring eve up on the **small 12.1.0-kvm** image; P3 fills the disk on first boot,
which is exactly the shrink precondition — no host-side disk surgery needed:

```sh
eden config add default
eden config set default --key=eve.tpm     --value=true
eden config set default --key=eve.accel   --value=true
eden config set default --key=eve.tag     --value=12.1.0       # small layout
eden config set default --key=eve.hostfwd --value='{"2222":"22","2223":"2223"}'
eden setup
eden start
eden eve onboard
```

(See the `kvm-to-k-conversion-testing` skill and `eden-bringup.sh` for the
swtpm-socket gotcha and a scripted bringup.)

> Single-tenant: only one eden runs per host. Do not start this against a host
> that already has an eden instance you care about.

This test does **not** bring eve up itself — it assumes a running, onboarded
device on the small image.

## Run

```sh
cd <this-eden-workspace>
RESIZE_EVE_VER=0.0.0-kvm-to-k-resize-<sha> \
  ./eden test ./tests/update_eve_image -e update_eve_image_kvm_to_k_resize -v debug
```

### Parameters

| Env var | Required | Default | Meaning |
|---------|----------|---------|---------|
| `RESIZE_EVE_VER` | **yes** | — | Version base of the local kvm-to-k-resize build, i.e. the part **before** `-kvm-amd64` / `-k-amd64`, e.g. `0.0.0-kvm-to-k-resize-<sha>`. Both the kvm hop and the k hop use this same base; the test appends `-kvm-<arch>` and `-k-<arch>`. |
| `RESIZE_EVE_REG` | no | `lfedge/eve` | The image **registry/repo namespace** (not a version). The full docker tag is `<RESIZE_EVE_REG>:<RESIZE_EVE_VER>-<hv>-<arch>`. `eden utils download eve-rootfs --eve-registry=<this>` extracts the rootfs from that **local** docker image — no registry push. `eve-build.sh` tags under `lfedge/eve`, so override only if you keep EVE images elsewhere. |
| `BRINGUP_EVE_VER` | no | `12.1.0` | Substring the running base release (`/run/eve-release`) must contain at Step 1, asserting the device is on the small bringup image and not already upgraded. Set it if you brought eve up on a small release other than 12.1.0. |

The escript does **not** install the small bringup image — that is whatever
`eve.tag` you set at eden setup. It only *asserts* the running base matches
`BRINGUP_EVE_VER` at Step 1.

## What each step watches

- **Step 1 `assert-running-version.sh` + `capture-partitions.sh assert-small`** —
  fail fast unless the device is on the expected small bringup release and the disk
  is genuinely small (ESP `< 256 MiB`, IMGA/IMGB `< 1 GiB`). P3 fills the disk, so
  the resizer's check returns `shrink`.
- **Step 8 `watch-conversion.sh`** — drives the conversion to completion (booted
  the `-k` version, active partition runs it) and logs device state/sub-state
  TRANSITIONS (running version, active slot, controller-reported `ZDEVICE_STATE`/
  sub-state, on-device `Converting`/`ConvertSubState`), tolerating the reboots the
  offline resize needs. Slot-agnostic (the grow relocates the active partition).
- **Step 9 `wait-for-volumemgr-ready.sh`** — the redeploy gate: volumemgr must
  finish its EVE-k content-store init before the redeploy, or the image is
  re-downloaded instead of reusing the retained ContentTree. Logs the readiness
  timeline.
- **Step 10 `capture-partitions.sh assert-grown` + `assert-seal-from-newlog.sh`** —
  ESP/IMGA/IMGB grew past the large floors and **P3 shrank**; and the
  repartition preserved the local TPM seal (post-hoc from `/persist/newlog`).
- **Step 11 `wait-for-app-running.sh` + `capture-blob-diagnostics.sh` +
  `snapshot-rcv-bytes.sh post-and-assert` + `wait-for-ssh-to-app.sh`** — the app
  comes online on EVE-k, **< 1 MiB** downloaded (blob reuse), and it is functional
  over the network (ssh to the eclient).

## Caveats / not-yet-validated

- The EVE-k first boot installs longhorn (kubevirt/CDI/longhorn/descheduler);
  helper budgets are sized for a slow rig but may need bumping.
- `assert-seal-from-newlog.sh` relies on the vaultmgr observability in the
  kvm-to-k-resize image (`VaultStatus.UnlockMethod` / the `unlocked: method=` log
  line). The bringup image (12.1.0) need not have it — the seal check keys on the
  conversion-capable kvm version's boots.
