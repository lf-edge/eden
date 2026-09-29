# kvm->k geometry matrix (`update_eve_image_kvm_to_k_geom`)

A geometry-only sibling of `update_eve_image_kvm_to_k_resize`. It proves that,
whatever the boot disk's **starting** GPT layout, driving an EVE-kvm -> EVE-k
base-OS upgrade ends with the full EVE-k target of **2+2+10+10 GiB**:

| partition | target |
|-----------|--------|
| ESP-A ("EFI System", PARTUUID `…30051`) | 2 GiB |
| ESP-B, reserved ("EFI System", PARTUUID `…30056`) | 2 GiB |
| IMGA | 10 GiB |
| IMGB | 10 GiB |

It deliberately skips apps, deferred content-tree deletes, blob-reuse, and the
TPM-seal check — those live in the resize/volmig tests. One escript, parameterized
by the starting release; the matrix runner runs it once per release.

## The one escript, driven by parameters

`update_eve_image_kvm_to_k_geom.txt` reads:

| env | meaning |
|-----|---------|
| `RESIZE_EVE_VER` (req) | conversion-capable build base, e.g. `0.0.0-kvm-to-k-resize-<sha8>` |
| `RESIZE_EVE_REG` | image namespace (default `lfedge/eve`) |
| `BRINGUP_EVE_VER` (req) | the release EVE was brought up on (asserted at Step 1) |
| `START_ESP_MIB` / `START_IMGA_MIB` / `START_IMGB_MIB` | expected start sizes (0 = skip that band check) |
| `START_HAS_ESPB` | 1 if the start image already has the reserved ESP-B |

Flow: assert START release + geometry → kvm→kvm hop onto `RESIZE_EVE_VER` (lands
the conversion code, geometry unchanged) → kvm→k (arms the offline repartition) →
`assert-final` = 2+2+10+10.

The two ESPs share the GPT label `"EFI System"`; `capture-geom.sh` tells them
apart by PARTUUID and by **count**, so `assert-final` requires **two** ~2 GiB
"EFI System" partitions, one carrying the reserved ESP-B UUID (`…30056`).

## Starting-release matrix

Sizes verified against each tag's `pkg/mkimage-raw-efi/make-raw`:

| release | ESP | ESP-B | IMGA | IMGB | what the convert must add |
|---------|-----|-------|------|------|---------------------------|
| **10.1.0**    | 36 MiB | — | 300 MiB | 300 MiB | grow ESP + both IMGx (+ shrink P3) + create ESP-B |
| **16.13.0**   | 2 GiB  | — | 4 GiB   | 4 GiB   | grow both IMGx (+ shrink P3) + create ESP-B |
| **17.0.0-rc1**| 2 GiB  | — | 10 GiB  | 10 GiB  | **only** create ESP-B |
| **17.1.0** (TBD, unreleased) | 2 GiB | 2 GiB | 10 GiB | 10 GiB | nothing (no-op; must stay 2+2+10+10) |

10.1.0 is the pre-512 MiB baseline (`ROOTFS_PART_SIZE=300 MiB`); 512 MiB first
shipped in 10.2.0. 2 GiB ESP first shipped in 16.13.0; 10 GiB IMGx in the 17.0.0
series; the reserved ESP-B is not in any release yet (build-time `make-raw` only).

## Status: e2e check for the ESP-B retrofit (RED until the EVE wiring lands)

This matrix is the end-to-end verification for the **ESP-B retrofit** (design:
`~/notes/esp-b-retrofit-3-design.md` — retrofit the reserved ESP-B onto old
pre-ESP-B disks during the kvm→k grow so a converted device is byte-identical to a
fresh EVE-k install: partition **#7**, GUID `…30056`, `ef00`, 2 GiB).

- **Library side (done + validated):** partitionresizer branch `esp-b-create` has
  the declarative create path (empty FAT32 by offset, ESP-B folded into the final
  GPT write, #7 reserved); diskfs PRs #21/#23/#414.
- **EVE side (in progress):** `pkg/storage-resizer` gains the dynamic 22/24 GiB
  budget (24 when ESP-B must be created) and wires `Apply` (select ESP-A by GUID
  `…30051`, create ESP-B `…30056` when absent) — branch `esp-b-create-resize` at
  `~/lf-edge/esp-b-create/eve`.

So `assert-final` (which requires two ~2 GiB "EFI System" partitions, one carrying
`…30056`) goes **green only when `RESIZE_EVE_VER` is a create-capable build**.
Against a build whose resizer still targets `spaceNeededBytes = 2+10+10` with no
ESP-B it is RED, correctly showing the not-yet-created state.

**Validated 2026-07-09 (all four starts GREEN)** in a multipass sandbox against
`0.0.0-esp-b-create-resize-nostress-6fb1e67c` (resizer `3eeaf245`): 10.1.0, 12.1.0,
16.13.0, 17.0.0-rc1 each converted to 2+2+10+10 with the reserved ESP-B created.
The sandbox suffices because this test boots no app guest (no depth-2 nested-KVM
wall). 17.1.0 additionally covers the already-converted no-op once such an image
exists.

## Running

Single-tenant eden — the runner takes over the host eden slot (destructive reset).

```sh
S=~/.claude/skills/kvm-to-k-conversion-testing/scripts
RESIZE_EVE_VER=0.0.0-kvm-to-k-resize-<sha8> bash "$S/run-kvm-to-k-geom-matrix.sh"
# add the unreleased ESP-B build once you have it built locally:
RESIZE_EVE_VER=... RELEASES="10.1.0 16.13.0 17.0.0-rc1 17.1.0" bash "$S/run-kvm-to-k-geom-matrix.sh"
```

Results (per-release logs + `summary.txt`) land in `~/kvm-to-k-geom-matrix-<utc>/`.
To run a single start by hand against an already-onboarded device, invoke the
escript directly (see `resize-fullrun.sh` for the `eden test … -e …` form) with
`BRINGUP_EVE_VER` + the `START_*` values from the table above.

## Operational notes (from the 2026-07-09 validation)

- **Watch budget scales with the IMGx-grow copy.** A start whose IMGA/IMGB must
  GROW (16.13.0: 4→10 GiB, ~8 GiB online-grow copy) can take ~30 min before the
  offline shrink + EVE-k boot even begin; on constrained I/O the whole run is
  ~70 min. The watch step is `exec -t 90m` for this reason. Tiny-IMGx starts
  (10.1.0/12.1.0) and no-grow starts (17.0.0-rc1) finish in ~32-40 min.
- **`RESIZE_EVE_VER` must be genuinely create-capable.** storage-init runs the
  storage-resizer *binary* pinned in `pkg/storage-init/Dockerfile`
  (`FROM lfedge/eve-storage-resizer:<hash>`), not the eve tree's Go source, so a
  build whose source has the create but pins a pre-feature binary produces
  2+10+10 (RED). Confirm the pin is create-capable before a green run.
- **10.1.0 boots + onboards fine** under current eden (was flagged as a risk; the
  2021→2026 kvm→kvm hop applied cleanly). No need to substitute 12.1.0.
- **17.1.0** needs a local build carrying both the ESP-B `make-raw` change and the
  conversion wiring; it is unreleased, so the runner skips it unless present.
