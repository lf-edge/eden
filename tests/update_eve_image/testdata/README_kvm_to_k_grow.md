# Eden setup for the kvm-to-k boot-disk repartition test — GROW (no-shrink) variant

`update_eve_image_kvm_to_k_grow.txt` drives the EVE-kvm → EVE-k in-field boot-disk
repartition (small → large GPT geometry) for the case where the boot disk already
has **≥ 22 GB of free space after the last partition**, so the resizer grows
ESP/IMGA/IMGB into that free tail **without shrinking `/persist`**. It is the
grow-only sibling of `update_eve_image_kvm_to_k_resize.txt` (the shrink variant,
where P3 fills the disk and must be shrunk first); the two escripts diff only on
those variant specifics.

The repartition runs **offline in storage-init**: baseosmgr arms a grow-only flag
and reboots; storage-init grows the partitions while `/persist` is unmounted (the
boot disk's GPT can't be re-read live while its rootfs is mounted). The test
asserts:

1. the disk **started small** and had a **≥ 22 GB free tail** (the `grow`
   precondition — `storage-resizer check`, fail-fast);
2. ESP/IMGA/IMGB **grew to the large geometry** and **P3 did NOT shrink**;
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

## eden bringup (TPM required) — small image THEN enlarge the disk

The vault assertions need a TPM, so eden must run with `eve.tpm=true` + swtpm. The
free tail must be created **after first boot**, not before: on its first boot
12.1.0's `storage-init.sh` runs `sgdisk --largest-new` and P3 would otherwise
swallow the enlarged disk. So boot small once, settle, stop, enlarge + relocate
the backup GPT, then restart:

```sh
# 1) bring up on the SMALL 12.1.0-kvm image with a TPM
eden config add default
eden config set default --key=eve.tpm     --value=true
eden config set default --key=eve.accel   --value=true
eden config set default --key=eve.tag     --value=12.1.0       # small layout
eden config set default --key=eve.hostfwd --value='{"2222":"22","2223":"2223"}'
eden setup
eden start
eden eve onboard
# let it settle: zedagent up, P3 created (fills the ORIGINAL ~28 GB disk)

# 2) stop eden and enlarge + fix the disk (host-side). qcow2 must be exposed as
#    a block device for sgdisk; needs sudo + the nbd module.
eden eve stop
IMG=dist/default-images/eve/live.img       # under EDEN_HOME: $EDEN_HOME/default-images/eve/live.img
cp -f "$IMG" "$IMG.prespare.bak"
qemu-img resize "$IMG" +24G                # >22 GB tail
sudo modprobe nbd max_part=16
sudo qemu-nbd --connect=/dev/nbd0 "$IMG"
sgdisk -v /dev/nbd0                        # expect: corrupt (secondary header not at end)
sudo sgdisk -e /dev/nbd0                   # relocate backup GPT + extend last-usable-LBA
sgdisk -v /dev/nbd0                        # expect: No problems found
sudo qemu-nbd --disconnect /dev/nbd0

# 3) resume — eden start ONLY (do NOT re-run `eden setup`; it regenerates live.img)
eden start
```

The free tail survives the later kvm→kvm hop and the settle reboots: once P3
exists the first-boot gate is false, so P3 is never re-grown. (See the
`kvm-to-k-conversion-testing` skill and `eden-bringup.sh` for the swtpm-socket
gotcha and a scripted bringup.)

> Single-tenant: only one eden runs per host. Do not start this against a host
> that already has an eden instance you care about.

This test does **not** bring eve up itself — it assumes a running, onboarded
device on the small image **with the free tail already in place**. Step 1 fails
fast (via `storage-resizer check`) if the tail is missing.

## Run

```sh
cd <this-eden-workspace>
RESIZE_EVE_VER=0.0.0-kvm-to-k-resize-<sha> \
  ./eden test ./tests/update_eve_image -e update_eve_image_kvm_to_k_grow -v debug
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

- **Step 1 `assert-running-version.sh` + `capture-partitions.sh assert-small` +
  `assert-free-tail.sh` + (post-kvm) `assert-check-grow.sh`** — fail fast unless
  the device is on the expected small bringup release, the disk is genuinely small
  (ESP `< 256 MiB`, IMGA/IMGB `< 1 GiB`), there is a **≥ 22 GB free tail**, and
  `storage-resizer check` reports `decision == grow`. The grow decision is the
  load-bearing precondition: if it is `shrink`, the bringup didn't create the tail
  and the test would silently become the shrink test.
- **Step 8 `watch-conversion.sh`** — drives the conversion to completion (booted
  the `-k` version, active partition runs it) and logs device state/sub-state
  TRANSITIONS (running version, active slot, controller-reported `ZDEVICE_STATE`/
  sub-state, on-device `Converting`/`ConvertSubState`), tolerating the reboots the
  offline resize needs. Slot-agnostic (the grow relocates the active partition).
- **Step 9 `wait-for-volumemgr-ready.sh`** — the redeploy gate: volumemgr must
  finish its EVE-k content-store init before the redeploy, or the image is
  re-downloaded instead of reusing the retained ContentTree. Logs the readiness
  timeline.
- **Step 10 `capture-partitions.sh assert-grown-no-shrink` + `assert-seal-from-newlog.sh`** —
  ESP/IMGA/IMGB grew past the large floors and **P3 did not shrink**; and the
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
