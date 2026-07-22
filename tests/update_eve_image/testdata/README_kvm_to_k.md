# EVE-kvm ↔ EVE-k conversion test suite

These escripts exercise the cross-flavor (EVE-kvm ↔ EVE-k) BaseOs upgrade and the
in-field **boot-disk repartition** it triggers: a fielded device installed with the
SMALL GPT geometry (36 MiB ESP, 512 MiB IMGA/IMGB, big P3) is converted, offline and
in the field, to the LARGE EVE-k geometry — ESP-A 2 GiB, a reserved **ESP-B** (GPT #7,
GUID …30056) 2 GiB, IMGA/IMGB 10 GiB — while preserving the deployed app, its
volume, cached OCI blobs, and the TPM-sealed vault across the geometry change.

The conversion runs **offline** in `storage-init` (the boot disk's GPT can't be
re-read live while its rootfs is mounted): the cross-flavor seam arms a shrink/grow
flag and reboots; `storage-init` grows ESP/IMGA/IMGB (shrinking the ext4 P3 first if
there is no free tail), then boots EVE-k on the enlarged geometry.

## Tests

| Escript | Validates | Start |
|---|---|---|
| `update_eve_image_kvm_to_k.txt` | The successful repartition (shrink **or** grow, single/two-disk/zfs), parameterized by env knobs. Asserts SMALL→LARGE geometry incl. ESP-B, `Converting` device state, TPM seal preserved, and cached-blob reuse on redeploy. | SMALL |
| `update_eve_image_kvm_to_k_refused.txt` | The declined (`insufficient`) path: conversion refused (too-full ext4, or ZFS persist), geometry unchanged, `BaseOsStatus.Error`, device stays manageable. | SMALL |
| `update_eve_image_kvm_to_k_geom.txt` | Geometry-only matrix: from every historical start geometry the conversion reaches the full 2+2+10+10 EVE-k target incl. ESP-B. No app/vault. | SMALL (per release) |
| `update_eve_image_kvm_to_k_volmig.txt` | App-volume migration without recreate (qcow2 → Longhorn PVC), for a VM app **and** a container app; data marker survives. | LARGE (`proceed`) |
| `update_eve_image_kvm_to_k_persist_wipe_restore.txt` | Fault: full `/persist` loss → identity restored from the `/config` backup, offline, incl. decrypt of a controller-encrypted credential from the restored ecdh cert. | SMALL |
| `update_eve_image_kvm_to_k_backup_corrupt_restore.txt` | Fault: partial corruption (backed-up files truncated) → recovery via the per-type validity gate + `.bak` fallback. | SMALL |
| `first_boot_evek_app_volume.txt` | Native EVE-k first boot (no conversion): an app volume already in config converges once cluster storage is ready (lf-edge/eve #6121). Asserts ESP-B present. | native EVE-k |
| `update_eve_image_cross_hv.txt` | Bare cross-HV upgrade (no volumes), both directions. | released |
| `update_eve_image_cross_hv_with_contenttree.txt` | A pre-staged ContentTree survives the cross-HV switch (blob reuse). | released |
| `update_eve_image_cross_hv_with_app_recreate.txt` | App + volume survive delete→redeploy across kvm→k and a controller-initiated device reboot (Longhorn re-attach). | released |

Only the `cross_hv*` escripts are registered in `eden.update_eve_image.tests.txt`
(they run on released images with no local build). The rest require a locally-built
conversion image and manual bringup, so they are **run explicitly**, not from the
`eden test` manifest — see below.

## Prerequisites

- A local EVE image **pair** built from the branch carrying the conversion code,
  tagged `<RESIZE_EVE_REG>:<RESIZE_EVE_VER>-kvm-<arch>` **and** `-k-<arch>` (both
  hypervisor flavors must be present locally).
- The SMALL bringup release (`BRINGUP_EVE_VER`, default `12.1.0`) available to
  `eden` (it is pulled if not local).
- `swtpm` + OVMF on the host, and eden configured with `eve.tpm=true` — the vault /
  TPM-seal assertions are meaningless without a TPM.
- The host eden slot **free**: eden is single-tenant, and every leg does its own
  destructive bringup (full reset).
- For `persist_wipe_restore` / `backup_corrupt_restore`: an `eden` binary that
  carries the `add-wireless` CLI (lf-edge/eden #1202), used to inject the encrypted
  credential whose offline decrypt is the load-bearing proof.

## Build the conversion image pair

Build EVE for both hypervisors from the branch that carries the conversion code
(e.g. `make HV=kvm eve` and `make HV=k eve` in the eve repo). Each build tags
`lfedge/eve:0.0.0-<branch>-<sha8>-<hv>-amd64`. Set `RESIZE_EVE_VER` to the shared
prefix (without the `-<hv>-<arch>` suffix), e.g.:

```sh
RESIZE_EVE_VER=0.0.0-resize-allprs-0c318dfa   # the -kvm and -k tags of this prefix must both exist
```

Commit before building so both flavors share a clean `0.0.0-<branch>-<sha8>` prefix
(a dirty tree injects a per-build `-dirty-<timestamp>` that differs between the kvm
and k builds).

## Run the repartition matrix

`run-kvm-to-k-tests.sh` drives the repartition + insufficient-space legs end to end:
for each disk topology it calls `prep-kvm-to-k-topology.sh <topology> --yes` (the
committed host-side bringup helper) and then runs the mapped escript with the right
knobs, collecting a PASS/FAIL summary.

```sh
RESIZE_EVE_VER=0.0.0-resize-allprs-<sha8> bash run-kvm-to-k-tests.sh
# subset:
ONLY=ext4-shrink,ext4-grow RESIZE_EVE_VER=0.0.0-resize-allprs-<sha8> bash run-kvm-to-k-tests.sh
```

Legs (topology → escript + knobs):

| Leg id | topology | escript + knobs |
|---|---|---|
| `ext4-shrink` | ext4 full, 1 disk | `kvm_to_k` `EXPECT_DECISION=shrink` |
| `ext4-grow` | ext4 + ≥22 GiB tail, 1 disk | `kvm_to_k` `EXPECT_DECISION=grow` |
| `twodisk-ext4` | ext4 on sdb, 2 disks | `kvm_to_k` `EXPECT_DECISION=grow DISK_TOPOLOGY=two-disk` |
| `twodisk-zfs` | zfs on sdb, 2 disks | `kvm_to_k` `EXPECT_DECISION=grow DISK_TOPOLOGY=two-disk` |
| `zfs-grow` | zfs on boot P3 + tail | `kvm_to_k` `EXPECT_DECISION=grow DISK_TOPOLOGY=zfs` |
| `ext4-toofull` | ext4 full (fill-driven) | `kvm_to_k_refused` `REFUSE_REASON=too-full` |
| `zfs-notail` | zfs on boot P3, no tail | `kvm_to_k_refused` `REFUSE_REASON=zfs` |

`prep-kvm-to-k-topology.sh <topology>` can also be run standalone to leave eden in a
given layout, then run the escript by hand.

## Other tests (own bringup)

These need a different start image or bringup than the repartition matrix, so run
them individually after bringing eden up as noted.

```sh
# App-volume migration — start LARGE on the conversion -kvm image (no repartition):
VOLMIG_EVE_VER=<ver> eden test tests/update_eve_image -e 'update_eve_image_kvm_to_k_volmig$' -v debug

# Geometry matrix — per starting release, set the start geometry it should have:
BRINGUP_EVE_VER=<rel> RESIZE_EVE_VER=<ver> \
  START_ESP_MIB=36 START_IMGA_MIB=512 START_IMGB_MIB=512 START_HAS_ESPB=0 \
  eden test tests/update_eve_image -e 'update_eve_image_kvm_to_k_geom$' -v debug

# Native EVE-k first boot — bring up directly on a -k image:
FBK_EVE_VER=<k-ver> eden test tests/update_eve_image -e 'first_boot_evek_app_volume$' -v debug

# Persist-wipe / backup-corruption restore — SMALL start; needs the add-wireless CLI:
RESIZE_EVE_VER=<ver> BRINGUP_EVE_VER=12.1.0 \
  eden test tests/update_eve_image -e 'update_eve_image_kvm_to_k_persist_wipe_restore$' -v debug
RESIZE_EVE_VER=<ver> BRINGUP_EVE_VER=12.1.0 \
  eden test tests/update_eve_image -e 'update_eve_image_kvm_to_k_backup_corrupt_restore$' -v debug

# Cross-HV family — ALT_HV selects the target flavor (also in the CI manifest):
ALT_HV=k ALT_EVE_VER=<ver> eden test tests/update_eve_image -e 'update_eve_image_cross_hv$' -v debug
```

## Env-knob reference

`update_eve_image_kvm_to_k.txt`:

| Knob | Default | Meaning |
|---|---|---|
| `RESIZE_EVE_VER` | *(required)* | version base of the local conversion build |
| `RESIZE_EVE_REG` | `lfedge/eve` | image registry/repo namespace |
| `BRINGUP_EVE_VER` | `12.1.0` | SMALL start release the device must be on at Step 1 |
| `EXPECT_DECISION` | `shrink` | `shrink` \| `grow` — Step-1 precondition + final geometry assert |
| `DISK_TOPOLOGY` | `single` | `single` \| `two-disk` \| `zfs` — data-preservation invariant |
| `POST_REBOOT_CHECK` | *(unset)* | non-empty ⇒ after the conversion, reboot the device from the controller and re-verify the app recovers |
| `FILL_PERSIST_GIB` | `33` | (shrink only) GiB to pre-fill `/persist` so the offline shrink has real blocks to relocate; `0` disables |
| `RELOCATE_CRITICAL_HIGH` | `0` | also relocate the identity-critical files into high blocks so the shrink must move them (soak) |
| `RELOCATE_STRICT` | `0` | hard-fail unless every critical file landed high |

Other escripts: `update_eve_image_kvm_to_k_refused.txt` adds `REFUSE_REASON`
(`too-full` \| `zfs`); `_volmig` uses `VOLMIG_EVE_VER`/`VOLMIG_EVE_REG` (LARGE
start); `_geom` uses `START_ESP_MIB`/`START_IMGA_MIB`/`START_IMGB_MIB`/`START_HAS_ESPB`
(+ `BRINGUP_EVE_VER`/`RESIZE_EVE_VER`); `_persist_wipe_restore` and
`_backup_corrupt_restore` add `SKIP_KVM_HOP`; `first_boot_evek_app_volume` uses
`FBK_EVE_VER`; the `cross_hv*` escripts use `ALT_HV`/`ALT_EVE_VER`/`ALT_EVE_REG`.

## Why the image sequence is small → kvm-hop → k

A released SMALL image lacks the conversion code, and only a SMALL-layout build
yields the SMALL start geometry the conversion must operate on. So the conversion
escripts always: (1) start SMALL on `BRINGUP_EVE_VER`; (2) BaseOs-update kvm→kvm
onto the conversion-capable build — same geometry, this only lands the code — then
settle the vault to a local TPM unlock; (3) BaseOs-update kvm→k — the cross-flavor
seam that arms the offline repartition. Getting this sequence wrong invalidates the
test.
