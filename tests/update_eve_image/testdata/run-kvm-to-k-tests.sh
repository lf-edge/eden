#!/bin/bash
# run-kvm-to-k-tests.sh — drive the EVE-kvm -> EVE-k boot-disk repartition test
# matrix end to end. For each disk topology, prep-kvm-to-k-topology.sh brings eden
# up in that layout on a SMALL start image, then the mapped escript performs the
# controller-driven kvm->k conversion and asserts the outcome. Verdicts are
# collected and the run continues on failure; a summary prints at the end.
#
# SCOPE. This drives the repartition + insufficient-space legs, which share the
# SMALL-start + local-build model whose bringup is the committed
# prep-kvm-to-k-topology.sh (single source of truth). The remaining kvm->k
# escripts need a different start image or bringup and are documented, with their
# exact invocation, under "Other tests" in README_kvm_to_k.md: the app-volume
# migration (large start), the geometry matrix (per-release starts), the
# native-EVE-k first-boot test, the persist-wipe / backup-corruption restore
# tests (plain small start), and the cross-HV family (released images).
#
# PREREQUISITES (see README_kvm_to_k.md for the full recipe):
#   * a local EVE image PAIR built from the conversion branch, tagged
#     <RESIZE_EVE_REG>:<RESIZE_EVE_VER>-{kvm,k}-<arch> (BOTH flavors present);
#   * the small bringup release (<BRINGUP_EVE_VER>, default 12.1.0) available;
#   * swtpm + OVMF on the host and eden configured with eve.tpm=true (the vault /
#     seal assertions are meaningless without a TPM);
#   * the host eden slot free — eden is single-tenant and every leg does its own
#     destructive bringup.
#
# ENV:
#   RESIZE_EVE_VER   (required) version base of the local conversion build, without
#                    the -<hv>-<arch> suffix, e.g. 0.0.0-resize-allprs-<sha8>.
#   RESIZE_EVE_REG   (default lfedge/eve) image registry/repo namespace.
#   BRINGUP_EVE_VER  (default 12.1.0) small-layout start release.
#   ONLY=a,b,...     run only these leg ids (comma-separated; ids below).
#   SKIP=a,b,...     skip these leg ids.
#   EDEN             eden binary (default: <workspace>/dist/bin/eden, else `eden`).
#
# LEG ids and their (topology -> escript + knobs) mapping — matches the
# prep-kvm-to-k-topology.sh header table:
#   ext4-shrink   ext4 full, 1 disk        -> kvm_to_k  EXPECT_DECISION=shrink
#   ext4-grow     ext4 +tail, 1 disk       -> kvm_to_k  EXPECT_DECISION=grow
#   twodisk-ext4  ext4 on sdb, 2 disks     -> kvm_to_k  EXPECT_DECISION=grow DISK_TOPOLOGY=two-disk
#   twodisk-zfs   zfs on sdb, 2 disks      -> kvm_to_k  EXPECT_DECISION=grow DISK_TOPOLOGY=two-disk
#   zfs-grow      zfs on boot P3, +tail    -> kvm_to_k  EXPECT_DECISION=grow DISK_TOPOLOGY=zfs
#   ext4-toofull  ext4 full (fill-driven)  -> kvm_to_k_refused  REFUSE_REASON=too-full
#   zfs-notail    zfs on boot P3, no tail  -> kvm_to_k_refused  REFUSE_REASON=zfs
set -uo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
PREP="$HERE/prep-kvm-to-k-topology.sh"
# testdata/ -> update_eve_image/ -> tests/ -> workspace root
WS="$(cd "$HERE/../../.." && pwd)"
EDEN="${EDEN:-$WS/dist/bin/eden}"
[ -x "$EDEN" ] || EDEN=eden
# prep-kvm-to-k-topology.sh and the escripts call bare `eden` and use paths
# relative to the workspace root, so run everything from $WS with dist/bin on PATH.
export PATH="$WS/dist/bin:$PATH"
TESTREL="tests/update_eve_image"

: "${RESIZE_EVE_VER:?set RESIZE_EVE_VER to your local conversion build, e.g. 0.0.0-resize-allprs-<sha8>}"
export RESIZE_EVE_VER
export RESIZE_EVE_REG="${RESIZE_EVE_REG:-lfedge/eve}"
export BRINGUP_EVE_VER="${BRINGUP_EVE_VER:-12.1.0}"

[ -f "$PREP" ] || { echo "FATAL: prep script not found: $PREP" >&2; exit 1; }

# id | topology | escript | extra env (space-separated VAR=VAL)
LEGS=(
  "ext4-shrink|ext4-shrink|update_eve_image_kvm_to_k|EXPECT_DECISION=shrink"
  "ext4-grow|ext4-grow|update_eve_image_kvm_to_k|EXPECT_DECISION=grow"
  "twodisk-ext4|twodisk-ext4|update_eve_image_kvm_to_k|EXPECT_DECISION=grow DISK_TOPOLOGY=two-disk"
  "twodisk-zfs|twodisk-zfs|update_eve_image_kvm_to_k|EXPECT_DECISION=grow DISK_TOPOLOGY=two-disk"
  "zfs-grow|zfs-grow|update_eve_image_kvm_to_k|EXPECT_DECISION=grow DISK_TOPOLOGY=zfs"
  "ext4-toofull|ext4-toofull|update_eve_image_kvm_to_k_refused|REFUSE_REASON=too-full"
  "zfs-notail|zfs-notail|update_eve_image_kvm_to_k_refused|REFUSE_REASON=zfs"
)

declare -A WANT NOWANT
if [ -n "${ONLY:-}" ]; then IFS=',' read -ra a <<<"$ONLY"; for x in "${a[@]}"; do WANT[$x]=1; done; fi
if [ -n "${SKIP:-}" ]; then IFS=',' read -ra a <<<"$SKIP"; for x in "${a[@]}"; do NOWANT[$x]=1; done; fi
selected() {
  [ -n "${NOWANT[$1]:-}" ] && return 1
  if [ -n "${ONLY:-}" ]; then
    [ -n "${WANT[$1]:-}" ] && return 0
    return 1
  fi
  return 0
}

# Reclaim the single-tenant slot before each leg. prep-kvm-to-k-topology.sh guards
# against a running eden and will NOT reset one, and the escript's final `eden eve
# reset` leaves the eden_* containers and the qemu/swtpm UP -- a surviving qemu
# holds a write-lock on the disk image and makes the next `eden setup` fail. So
# stop eden, drop the eden_* containers + volumes (stale adam certs otherwise
# reject the new device), kill+verify any qemu/swtpm still bound to eden.root, and
# drop the prior leg's disk images so `eden setup` regenerates a fresh topology.
# Mirrors the validated bringup-grow.sh teardown.
free_slot() {
  "$EDEN" stop >/dev/null 2>&1 || true
  docker rm -fv eden_adam eden_redis eden_registry eden_eserver >/dev/null 2>&1 || true
  docker volume rm -f eden_adam_volume eden_redis_volume eden_eserver_volume eden_registry_volume >/dev/null 2>&1 || true
  local root
  root=$("$EDEN" config get default --key eden.root 2>/dev/null | grep -vi 'level=' | tr -d '\r' | tail -1)
  root="${root:-${EDEN_HOME:-$HOME/.e166o}}"
  for p in $(pgrep -f "[q]emu-system-x86.*$root" 2>/dev/null) $(pgrep -f "[s]wtpm.*$root" 2>/dev/null); do
    kill "$p" 2>/dev/null; sleep 1; kill -9 "$p" 2>/dev/null || true
  done
  sleep 1
  pgrep -f "[q]emu-system-x86.*$root" >/dev/null 2>&1 && echo "WARN: qemu on $root survived teardown (next setup may write-lock)" >&2
  rm -f "$root"/default-images/eve/live.img "$root"/default-images/eve/live.raw* "$root"/default-images/eve/eve-disk-*.qcow2 2>/dev/null || true
}

now() { date -u +%Y-%m-%dT%H:%M:%SZ; }
declare -A VERDICT
ORDER=()

for spec in "${LEGS[@]}"; do
  IFS='|' read -r id topo escript extra <<<"$spec"
  ORDER+=("$id")
  if ! selected "$id"; then VERDICT[$id]="SKIP(deselected)"; continue; fi
  echo
  echo "########## [$id] $topo -> $escript ($extra) @ $(now) ##########"
  free_slot
  if ! ( cd "$WS" && bash "$PREP" "$topo" --yes ); then
    VERDICT[$id]="FAIL(prep)"
    echo "########## [$id] FAIL(prep) @ $(now) ##########"
    continue
  fi
  # $extra is intentionally word-split into VAR=VAL args for env
  # shellcheck disable=SC2086
  if ( cd "$WS" && env $extra "$EDEN" test "$TESTREL" -e "${escript}\$" -v debug ); then
    VERDICT[$id]=PASS
  else
    VERDICT[$id]="FAIL(escript rc=$?)"
  fi
  echo "########## [$id] ${VERDICT[$id]} @ $(now) ##########"
done

echo
echo "===================== SUMMARY @ $(now) ====================="
echo "  image: ${RESIZE_EVE_REG}:${RESIZE_EVE_VER}-{kvm,k}   bringup: ${BRINGUP_EVE_VER}"
npass=0; nfail=0; nskip=0
for id in "${ORDER[@]}"; do
  v="${VERDICT[$id]:-?}"
  printf '  %-14s %s\n' "$id" "$v"
  case "$v" in PASS) npass=$((npass+1));; FAIL*) nfail=$((nfail+1));; SKIP*) nskip=$((nskip+1));; esac
done
echo "  ----"
echo "  PASS=$npass  FAIL=$nfail  SKIP=$nskip"
[ "$nfail" -eq 0 ]
