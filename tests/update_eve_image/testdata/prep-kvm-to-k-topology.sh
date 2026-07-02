#!/bin/bash
# prep-kvm-to-k-topology.sh — bring eden up in one of the disk topologies the
# kvm->k repartition tests need, BEFORE running the escript. Host-side helper,
# NOT an escript (it is never enumerated by `eden test`).
#
# The conversion tests always start on a SMALL, old bringup image; what differs
# between scenarios is the /persist filesystem (ext4 vs ZFS) and disk layout
# (single vs two disks), which determine the storage-resizer `check` decision and
# therefore which escript applies:
#
#   topology       persist        layout            check decision   escript / knobs                          status
#   -------------  -------------  ----------------  ---------------  --------------------------------         ------
#   ext4-shrink    ext4, full     1 disk            shrink           kvm_to_k.txt EXPECT_DECISION=shrink       ok
#   ext4-grow      ext4, +tail    1 disk            grow             kvm_to_k.txt EXPECT_DECISION=grow         ok (README recipe)
#   twodisk-zfs    zfs on sdb     2 disks           grow             kvm_to_k.txt ... DISK_TOPOLOGY=two-disk   VERIFIED 2026-06-22
#   ext4-toofull   ext4, full     1 disk            insufficient     kvm_to_k_refused.txt REFUSE_REASON=too-full  VERIFIED 2026-07-01
#   zfs-grow       zfs,  +tail    1 disk            grow             kvm_to_k.txt ... DISK_TOPOLOGY=zfs        VERIFIED 2026-06-25 (notail base)
#   zfs-notail     zfs, full      1 disk            insufficient     kvm_to_k_refused.txt REFUSE_REASON=zfs    VERIFIED 2026-06-25
#
# Decision logic is from pkg/storage-resizer (decide()/evaluate()): shrink applies
# ONLY to an ext4 /persist on the boot disk; a ZFS persist or a persist on another
# disk can only get room from the boot disk's free tail, else `insufficient`.
# (`grow` is documented as "also the multi-disk / ZFS-persist case".)
#
# HOW ZFS /persist IS BUILT (verified empirically in an eden Multipass sandbox):
#   * SINGLE-disk ZFS-on-boot (VERIFIED 2026-06-25): the `--grub-options` route DOES
#     work; the earlier "doesn't work" was a misdiagnosis. Two real reasons it had
#     been failing:
#       1. setupConfigDir (pkg/openevec/eden.go) only calls GenerateEveCerts — which
#          is what WRITES grub.cfg — when CertsDir has NO root-certificate.pem. On a
#          reused context the certs already exist, so `eden setup --grub-options ...`
#          SILENTLY skips grub.cfg generation and the token never lands. `--force`
#          does NOT override this. Fix: `rm -rf dist/default-certs` (+ adam/redis
#          volumes) before setup so the grub.cfg is regenerated. (A fresh EDEN_HOME,
#          as in the run-eden-test skill's ZFS recipe, has the same effect.)
#       2. The live.img has NO pre-baked P3 (only EFI/IMGA/IMGB/CONFIG); storage-init
#          CREATES P3 on first boot and honors P3_FS_TYPE_DEFAULT=zfs (set by the
#          token at storage-init.sh:113) -> P3 formatted ZFS on the boot disk.
#     Verified result: /proc/cmdline carries eve_install_zfs_with_raid_level,
#     /run/eve.persist_type=zfs, zpool ONLINE on sda9 (boot disk P3), /persist mounted.
#     The P3 takes the whole free tail (largest-new) -> no tail left -> this is the
#     zfs-notail (insufficient) base; zfs-grow adds a tail AFTER it (see that case).
#   * The canonical eden MULTI-disk ZFS recipe (run-eden-test skill) is
#     `eve.disks=4 eve.disk=16384` + the same grub-option; that puts the pool on the
#     extra blank disks rather than the boot P3.
#   * `eden setup --installer` requires the `general` devmodel and is NOT
#     qemu-managed (only ZedVirtual-4G gets a qemu config + `eden start`), so eden
#     cannot run an installer-built node itself.
#   * Working route (two-disk): boot the live node with a blank second disk, have
#     EVE ITSELF create the `persist` zpool on sdb (feature-correct — EVE's own
#     zfs), `zfs set -u mountpoint=/persist`, export it, then OFFLINE delete the
#     boot disk's P3 so storage-init's no-P3 branch does `zpool import -f persist`
#     and mounts /persist from sdb. Verified: persist_type=zfs, no P3 on boot, pool
#     ONLINE on sdb, /persist mounted.
#
# The additional disks come from eve.disks (count), each sized at eve.disk MB
# (pkg/openevec/eden.go:101 — extra disks inherit ImageSizeMB). NOTE: the live.img
# boot disk is the DOWNLOADED size (~28 GiB), not eve.disk — eve.disk only sizes
# the EXTRA disks. So `eve.disks=1` gives a ~28 GiB boot disk + a 28 GiB (or
# eve.disk-MB) second disk.
#
# The ext4 grow tail is created the way README_kvm_to_k_grow.md describes: boot
# small once (storage-init creates P3 at the original size), stop, enlarge the
# qcow2 + `sgdisk -e` to add the free tail, restart.
#
# Usage:
#   ./prep-kvm-to-k-topology.sh <topology> [--yes]
# eden-touching stages are gated behind a check that no OTHER eden is already
# running on this host (single-tenant — see CLAUDE.md).

set -uo pipefail

TOPOLOGY="${1:-}"
ASSUME_YES=0
[ "${2:-}" = "--yes" ] && ASSUME_YES=1

BRINGUP_TAG="${BRINGUP_EVE_VER:-12.1.0}"   # small-layout released image
GROW_TAIL_GB="${GROW_TAIL_GB:-24}"         # >22 GiB so check decides grow
EVE_DISK_MB="${EVE_DISK_MB:-32768}"        # size of the EXTRA disk(s) (not the boot disk)
SSH_TRIES="${SSH_TRIES:-40}"               # ssh-up poll attempts (x12s)
TOOFULL_BOOT_GB="${TOOFULL_BOOT_GB:-64}"   # ext4-toofull boot-disk size: big enough that an
                                           # EMPTY P3 would shrink fine, so the refusal is
                                           # genuinely fill-driven, not "disk too small".
FILL_PCT="${FILL_PCT:-70}"                 # ext4-toofull: fill /persist to this used% so
                                           # used > maxFull(90%) x (P3-22G) => check insufficient.

die() { echo "FATAL: $*" >&2; exit 1; }
note() { echo "=== $* ==="; }

usage() { sed -n '2,60p' "$0" | sed 's/^# \{0,1\}//'; exit 2; }
[ -n "$TOPOLOGY" ] || usage

# --- single-tenant guard: refuse if another eden is already up -----------------
guard_no_other_eden() {
    local busy=0
    docker ps --format '{{.Names}}' 2>/dev/null | grep -qE '^eden_(adam|redis|eserver|registry)$' && busy=1
    pgrep -f 'qemu-system.*-drive' >/dev/null 2>&1 && busy=1
    if [ "$busy" = 1 ]; then
        echo "An eden instance appears to be running on this host (eden_* containers or qemu)." >&2
        echo "eden is single-tenant; this script would disturb it. Stop it first or use a" >&2
        echo "dedicated host / the eden-vm-sandbox skill. Refusing." >&2
        exit 1
    fi
}

confirm() {
    [ "$ASSUME_YES" = 1 ] && return 0
    read -r -p "$1 [y/N] " a
    case "$a" in y|Y|yes) return 0 ;; *) echo "aborted."; exit 1 ;; esac
}

image_dir() {
    if [ -n "${EDEN_HOME:-}" ] && [ -d "$EDEN_HOME/default-images/eve" ]; then
        echo "$EDEN_HOME/default-images/eve"; return
    fi
    [ -d dist/default-images/eve ] && { echo dist/default-images/eve; return; }
    die "could not locate the eve image dir (set EDEN_HOME or run from the eden workspace root)"
}

base_config() {
    note "base eden config (TPM + accel + small bringup tag $BRINGUP_TAG)"
    eden config add default
    eden config set default --key=eve.tpm     --value=true
    eden config set default --key=eve.accel   --value=true
    eden config set default --key=eve.tag     --value="$BRINGUP_TAG"
    eden config set default --key=eve.hostfwd --value='{"2222":"22","2223":"2223"}'
}

eve_ssh() { eden eve ssh -- "$1" 2>/dev/null | grep -v 'level=fatal'; }

wait_ssh() {
    note "waiting for EVE ssh"
    local i
    for i in $(seq 1 "$SSH_TRIES"); do
        timeout 8 eden eve ssh -- "true" 2>/dev/null && { echo "ssh up"; return 0; }
        sleep 12
    done
    die "EVE ssh did not come up after $((SSH_TRIES*12))s"
}

# Enlarge a qcow2 boot disk by GROW_TAIL_GB and relocate the backup GPT so the
# free tail is usable (README_kvm_to_k_grow.md recipe). eden must be stopped.
add_free_tail() {
    local img="$1"
    [ -f "$img" ] || die "boot image not found: $img"
    note "enlarging $img by +${GROW_TAIL_GB}G and relocating backup GPT"
    cp -f "$img" "$img.prespare.bak"
    qemu-img resize "$img" "+${GROW_TAIL_GB}G"
    sudo modprobe nbd max_part=16
    sudo qemu-nbd --connect=/dev/nbd0 "$img"; sleep 1
    sgdisk -v /dev/nbd0 || true       # expect: corrupt (secondary header not at end)
    sudo sgdisk -e /dev/nbd0          # relocate backup GPT + extend last-usable-LBA
    sgdisk -v /dev/nbd0               # expect: No problems found
    sudo qemu-nbd --disconnect /dev/nbd0
}

# Grow the boot disk to an ABSOLUTE size BEFORE first boot and relocate the backup
# GPT so storage-init's first-boot P3 carve fills the whole disk (NO tail — the
# ext4-toofull topology). Unlike add_free_tail this is sudo-free: sgdisk on a raw
# *file* is plain file I/O, and a qcow2 is round-tripped through a sparse raw
# intermediate (qemu-img convert is sparse-aware, so cheap). Call while eden is
# stopped, between `eden setup` and the first `eden start`.
grow_boot_no_tail() {
    local img="$1" size_gb="$2" fmt
    [ -f "$img" ] || die "boot image not found: $img"
    fmt=$(qemu-img info "$img" | sed -ne 's/^file format: //p')
    note "growing $img to ${size_gb}G (no tail; fmt=$fmt) + relocating backup GPT"
    qemu-img resize "$img" "${size_gb}G" || die "boot resize failed"
    if [ "$fmt" = raw ]; then
        sgdisk -e "$img" || die "sgdisk -e failed"
        sgdisk -v "$img" || die "sgdisk -v failed"
    else
        local raw="$img.raw.tmp"
        qemu-img convert -f qcow2 -O raw "$img" "$raw" || die "qcow2->raw failed"
        sgdisk -e "$raw" || die "sgdisk -e on raw failed"
        sgdisk -v "$raw" || die "sgdisk -v on raw failed"
        qemu-img convert -f raw -O qcow2 "$raw" "$img" || die "raw->qcow2 failed"
        rm -f "$raw"
    fi
}

# Fill the ext4 /persist to FILL_PCT% used with random-content files so the
# storage-resizer shrink check (which reads the ext4 filesystem's used blocks
# directly — NOT volumemgr's RemainingSpace, so no declared volume is needed)
# can't free 22G within --max-full => decision `insufficient` (reason=too-full).
# Gauges on `df` used% (what the resizer reads), leaving ~(100-FILL_PCT)% free so
# the later kvm->kvm hop's rootfs blob still fits.
fill_persist() {
    local dir=/persist/resize-fill i usedpct
    note "filling /persist to ${FILL_PCT}% used (random content)"
    eve_ssh "eve exec pillar mkdir -p $dir"
    for i in $(seq 1 400); do
        usedpct=$(eve_ssh "eve exec pillar sh -c 'df --output=pcent /persist | tail -1 | tr -dc 0-9'" | tr -d '\r' | tail -1)
        usedpct=${usedpct:-0}
        echo "  /persist used=${usedpct}% (target ${FILL_PCT}%)"
        [ "$usedpct" -ge "$FILL_PCT" ] 2>/dev/null && { echo "  reached ${usedpct}%"; break; }
        eve_ssh "eve exec pillar sh -c 'dd if=/dev/urandom of=$dir/f$i bs=1M count=1024 conv=fsync 2>/dev/null; sync'"
    done
    eve_ssh "eve exec pillar sh -c 'df -h /persist'"
}

# Create the EVE persist zpool on a block device FROM INSIDE the running EVE, so
# the pool's feature flags match EVE's own zfs and storage-init can import it.
# Mirrors storage-init.sh's create; `zfs set -u` sets the mountpoint WITHOUT
# mounting now (avoids clobbering the live ext4 /persist), so the next boot's
# `zpool import` mounts it at /persist. $1 = device (e.g. /dev/sdb).
eve_make_persist_pool() {
    local dev="$1"
    # Build the pool to MATCH a real EVE-ZFS install — a faithful mirror of
    # pkg/installer/install prepare_mounts_and_zfs_pool (the installer, not storage-init,
    # is what lays down a multi-disk ZFS persist): same pool flags, PLUS persist/reserved
    # (refreservation = available/5), primarycache=metadata, and the non-clustered
    # persist/snapshots dataset. The bringup base is kvm => non-clustered shape. Child
    # datasets are created with `-u` (do not mount now) because the live ext4 /persist is
    # still mounted during this build; they mount on the later storage-init import.
    note "EVE creating persist zpool on $dev (installer-faithful: +reserved +primarycache +snapshots)"
    eve_ssh "eve exec pillar sh -c \"zpool labelclear -f $dev 2>/dev/null; \
        zpool create -f -m none -o feature@encryption=enabled -O atime=off -O overlay=on persist $dev && echo CREATED\"" \
        | grep -q CREATED || die "zpool create failed on $dev"
    # refreservation = available/5, computed exactly as the installer does (bytes -> MiB, /5)
    local avail_bytes resv_mib
    avail_bytes=$(eve_ssh "eve exec pillar zfs get -o value -Hp available persist" | grep -oE '^[0-9]+$' | head -1)
    [ -n "$avail_bytes" ] || die "could not read pool available bytes"
    resv_mib=$(( avail_bytes / 1024 / 1024 / 5 ))
    note "persist/reserved refreservation = ${resv_mib}m (= available/5)"
    eve_ssh "eve exec pillar sh -c \"\
        zfs create -u -o refreservation=${resv_mib}m persist/reserved && \
        zfs set -u mountpoint=/persist persist && \
        zfs set primarycache=metadata persist && \
        zfs create -u -o mountpoint=/persist/containerd/io.containerd.snapshotter.v1.zfs persist/snapshots && \
        zpool export persist && echo POOL_READY\"" | grep -q POOL_READY \
        || die "failed to set up/export persist pool on $dev"
}

# Offline-delete the boot disk's P3 (partition 9) so storage-init takes the no-P3
# -> `zpool import persist` path. eden must be stopped. $1 = boot qcow2.
delete_boot_p3() {
    local img="$1"
    note "offline-deleting boot P3 (partition 9) from $img"
    sudo modprobe nbd max_part=16
    sudo qemu-nbd --connect=/dev/nbd0 "$img"; sleep 1
    sudo sgdisk -d 9 /dev/nbd0 || die "sgdisk -d 9 failed"
    sudo sgdisk -p /dev/nbd0 | grep -qE 'P3|persist' && { sudo qemu-nbd --disconnect /dev/nbd0; die "P3 still present after delete"; }
    sudo qemu-nbd --disconnect /dev/nbd0
    echo "boot P3 deleted"
}

wait_settled() {
    note "onboard + settle (P3/persist created)"
    eden eve onboard || true
    eden status || true
    confirm "Has the device onboarded and settled (P3/persist created)?"
}

case "$TOPOLOGY" in

    ext4-shrink)
        # Single ext4 disk, P3 fills the disk (no tail) — the default install.
        guard_no_other_eden
        base_config
        note "eden setup (live.img, single ext4 disk, no tail)"
        eden setup; eden start; eden eve onboard
        echo "READY: ext4-shrink. Run: EXPECT_DECISION=shrink ... kvm_to_k"
        ;;

    ext4-grow)
        # Single ext4 disk + >=22G free tail. live.img bringup, then enlarge.
        guard_no_other_eden
        base_config
        note "eden setup (live.img, single ext4 disk)"
        eden setup; eden start
        wait_settled
        note "stopping eden to enlarge the boot disk"
        eden eve stop; sleep 5
        add_free_tail "$(image_dir)/live.img"
        note "resuming (eden start only — do NOT re-run eden setup; --force regenerates live.img)"
        eden start
        echo "READY: ext4-grow. Run: EXPECT_DECISION=grow ... kvm_to_k"
        ;;

    twodisk-zfs)
        # VERIFIED 2026-06-22 (eden sandbox). Two disks: sda boot (P3 deleted =>
        # free tail), sdb ZFS /persist. EVE builds the pool; we delete boot P3.
        guard_no_other_eden
        base_config
        eden config set default --key=eve.disks --value=1   # adds sdb
        eden config set default --key=eve.disk  --value="$EVE_DISK_MB"  # sizes sdb
        note "eden setup + start + onboard (sda boot ext4 P3, sdb blank)"
        eden setup; eden start; eden eve onboard
        wait_ssh
        eve_make_persist_pool /dev/sdb
        note "stopping eden to delete the boot P3"
        eden eve stop; sleep 6
        delete_boot_p3 "$(image_dir)/live.img"
        note "restart — storage-init no-P3 branch imports persist from sdb"
        eden start
        wait_ssh
        note "verify"
        eve_ssh "echo persist_type=\$(cat /run/eve.persist_type); eve exec pillar sh -c \"zpool status persist; df -h /persist\""
        echo "READY: twodisk-zfs. Run: EXPECT_DECISION=grow DISK_TOPOLOGY=two-disk ... kvm_to_k"
        ;;

    ext4-toofull)
        # Single ext4 disk grown to TOOFULL_BOOT_GB so an EMPTY P3 could shrink to
        # free 22G; then /persist is filled to FILL_PCT% so the shrink CAN'T free
        # 22G within --max-full => check `insufficient` (reason=too-full). Refused
        # case. VERIFIED end-to-end on host eden 2026-07-01 (e2e testplan row C):
        # check `decision=insufficient persistType=ext4 shrinkApplicable=true
        # shrink.ok=false`, kvm->k declined with BaseOsStatus.Error
        # "conversion not possible: persist is too full to free the needed space".
        guard_no_other_eden
        base_config
        eden config set default --key=eve.disks --value=0   # single boot disk
        note "eden setup (live.img, single ext4 disk)"
        eden setup
        grow_boot_no_tail "$(image_dir)/live.img" "$TOOFULL_BOOT_GB"
        eden start; eden eve onboard
        wait_ssh
        # A ZFS persist here would be the wrong topology (that is zfs-notail).
        ptype=$(eve_ssh 'cat /run/eve.persist_type' | tr -d '\r' | tail -1)
        [ "$ptype" = ext4 ] || die "persist_type='$ptype' (expected ext4 for ext4-toofull)"
        fill_persist
        echo "READY: ext4-toofull. Run: REFUSE_REASON=too-full ... kvm_to_k_refused"
        ;;

    zfs-grow|zfs-notail)
        # SINGLE-disk ZFS /persist (persist == the boot disk's P3 as ZFS).
        # VERIFIED 2026-06-25. The token is delivered via grub.cfg, which eden only
        # (re)writes when CertsDir has no root-certificate.pem (setupConfigDir gate) —
        # so we MUST wipe dist/default-certs (and adam/redis volumes) before setup,
        # else `eden setup --grub-options ...` silently skips grub.cfg and the token
        # never lands. live.img has no pre-baked P3, so storage-init creates it on
        # first boot and formats it ZFS because the token sets P3_FS_TYPE_DEFAULT=zfs
        # (storage-init.sh:113). Result: persist_type=zfs, zpool ONLINE on sda9.
        guard_no_other_eden
        base_config
        eden config set default --key=eve.disks --value=0   # single boot disk
        note "wipe certs + adam/redis volumes so eden setup regenerates grub.cfg"
        eden stop || true
        rm -rf "$(image_dir)/live.img" "$(image_dir)/live.raw.qcow2" dist/default-certs
        docker volume rm eden_adam_volume eden_redis_volume 2>/dev/null || true
        note "eden setup with the grub-option (canonical bare token, set_global form)"
        # $dom0_extra_args is a grub variable expanded at boot; it must reach grub
        # literally, so keep the single quotes and do not let the shell expand it.
        # shellcheck disable=SC2016
        eden setup --grub-options 'set_global dom0_extra_args "$dom0_extra_args eve_install_zfs_with_raid_level "'
        [ -f dist/default-certs/grub.cfg ] || die "grub.cfg not written — certs were not wiped?"
        eden start; eden eve onboard
        wait_ssh
        note "confirm the cmdline took effect"
        eve_ssh "echo cmdline=\$(cat /proc/cmdline); echo persist_type=\$(cat /run/eve.persist_type)"
        # Expect: cmdline contains eve_install_zfs_with_raid_level AND persist_type=zfs.
        if [ "$TOPOLOGY" = zfs-grow ]; then
            note "zfs-grow: add a >=22G free tail after the ZFS P3"
            eden eve stop; sleep 5
            add_free_tail "$(image_dir)/live.img"   # ZFS partition is not auto-grown -> leaves a tail
            eden start
            echo "READY(if persist_type=zfs above): zfs-grow. Run: EXPECT_DECISION=grow DISK_TOPOLOGY=zfs ... kvm_to_k"
        else
            echo "READY(if persist_type=zfs above): zfs-notail. Run: REFUSE_REASON=zfs ... kvm_to_k_refused"
        fi
        ;;

    *)
        echo "unknown topology: $TOPOLOGY" >&2
        usage
        ;;
esac
