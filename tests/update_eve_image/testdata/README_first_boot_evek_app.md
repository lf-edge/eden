# First-boot EVE-k app-volume test (`first_boot_evek_app_volume`)

Verifies **lf-edge/eve #6121** (merged to master 2026-07-16): on a **fresh EVE-k**
first boot, an app instance whose volume is requested **before cluster storage
(k3s + longhorn + CDI) is ready** must **converge**, not wedge. Before #6121 the
app volume could park indefinitely at `CREATING_VOLUME` (or, depending on where
the storage pipeline stuck, `INITIAL` / `LOADED`). #6121 defers EVE-k volume
creation until `ClusterStorageReadyForVolumes` (longhorn StorageClass + the CDI
control-plane Deployments), retries transient cluster-volume failures, and makes
`CreatePVC` idempotent.

This is a **native EVE-k boot** — NOT a kvm→k conversion. No repartition, no
cross-flavor upgrade, no vault settle.

## What it does

1. **Bringup (manual)** boots EVE directly on a `-k` (kubevirt) image and onboards
   it — this is the device's *first boot*. No app is deployed yet.
2. The **escript** then, as its first action, deploys an Ubuntu VM app (with a
   disk volume) — seconds after onboard, while EVE-k's cluster storage is still
   ~12–15 min from ready. So the app volume sits in the config through the entire
   no-storage window that #6121 protects.
3. It waits for storage to come up (`volumemgr` Initialized, longhorn
   StorageClass), then asserts the app volume was **not wedged** and the app
   reaches **RUNNING**, logging the `LAST_STATE(EVE)` progression so you can see it
   advance out of `CREATING_VOLUME`/`INITIAL`/`LOADED` into activation.
4. Strong liveness: key-based SSH into the guest (not just VMI-phase RUNNING).

**Ordering note:** eden's `pod deploy` targets an already-onboarded device (it
resolves the device UUID from the local eden context), so onboard must precede
deploy. Deploying immediately after a fresh onboard is the faithful reproduction —
the cluster is ~15 min from ready, so the volume is requested from the start of
EVE-k's operational life.

## Image

Uses a `-k` build carrying the #6121 stack. The most recent **pre-merge #6121**
build is the `volumemgr-cdi-retry` branch (identical commit subjects to master's
`5b53d02f7`…`3d35218e9`); its newest local `-k` image auto-discovers. To run
against a genuine **master** `-k` once one is built, pass `TAG=`/`FBK_EVE_VER=`.

## Run (host eden)

```sh
S=~/.claude/skills/kvm-to-k-conversion-testing/scripts
# 1) bringup (takes over the host eden slot; destructive reset)
bash "$S/firstboot-evek-bringup.sh"            # or TAG=0.0.0-master-<sha8> ...
# 2) run the escript
bash "$S/firstboot-evek-run.sh"                # or FBK_EVE_VER=<ver> ...
```

`firstboot-evek-run.sh` sets `FBK_EVE_VER`, checks EVE→adam connectivity, and runs
`eden.escript.test -test.run TestEdenScripts/first_boot_evek_app_volume`.

## Timing

Budgets follow the kvm→k tests (EVE-k cluster bring-up dominates): volumemgr
Initialized ~15–25 min into first boot, longhorn StorageClass shortly after, app
RUNNING once storage is ready. A healthy host run is ~25–40 min; the escript's
`-t 45m` waits are ceilings. The reboot watchdog is a 75 min safety ceiling
(a fresh EVE-k boot performs no reboots of its own).

## Reading a failure

- `FAIL[volume-wedge]: app stuck at <state> (below activation)` — the app never
  activated; `<state>` names where the storage pipeline stuck (`CREATING_VOLUME`,
  `INITIAL`, `LOADED`, …). This is the #6121 regression signal. Any
  `VolumeStatus`/`ContentTreeStatus` `Error` is dumped alongside.
- A stall in `wait-for-longhorn-sc.sh` means cluster storage itself never came up
  (a cluster-bringup problem, not the volume-gate path under test).
