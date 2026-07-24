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
#   twodisk-ext4   ext4 on sdb    2 disks           grow             kvm_to_k.txt ... DISK_TOPOLOGY=two-disk   VERIFIED 2026-07-07
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
#          does NOT override this. Fix: wipe the certs dir (certs_dir(), under
#          eden.root) + adam/redis volumes before setup so grub.cfg is regenerated.
#          (A fresh EDEN_HOME,
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
# eve.disk (ImageSizeMB) sizes BOTH the boot live.img AND every extra disk: eden
# passes it to the EVE image's `live <MB>` generator (pkg/utils/downloaders.go
# genEVELiveImage) for the boot disk, and to CreateDisk (pkg/openevec/eden.go:103)
# for each of the eve.disks extra disks. So base_config sets eve.disk=64 GiB for a
# single 64 GiB boot disk (the EVE-k/longhorn floor); the two-disk legs override it
# to 32768 to get a 32 GiB boot + 32 GiB sdb (=> 64 total).
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

# Detach stdin from any controlling terminal, mirroring the escript engine
# (cmd.Stdin = strings.NewReader("")): `timeout eden eve ssh` runs ssh in a
# background process group, where a tty stdin would SIGTTIN-stop it forever.
exec </dev/null

BRINGUP_TAG="${BRINGUP_EVE_VER:-12.1.0}"   # small-layout released image
BOOT_DISK_MB="${BOOT_DISK_MB:-65536}"      # single boot-disk size: 64 GiB meets the EVE-k/
                                           # longhorn floor. No-tail legs (shrink, toofull,
                                           # zfs-notail) use it directly; grow/two-disk legs
                                           # override eve.disk to 32768 (=> 32 GiB boot + 32 GiB
                                           # tail, or + 32 GiB sdb = 64 total).
GROW_TAIL_GB="${GROW_TAIL_GB:-32}"         # free tail for the single-disk grow legs, added after
                                           # a 32 GiB boot: >22 GiB so check decides grow and
                                           # 32+32 = the 64 GiB total floor.
EVE_DISK_MB="${EVE_DISK_MB:-32768}"        # grow/two-disk boot + extra-disk size (32 GiB)
TWODISK_BOOT_GB="${TWODISK_BOOT_GB:-32}"   # twodisk-ext4 boot-disk size (matches EVE_DISK_MB):
                                           # P3 created full then deleted => >=22 GiB free tail.
EDEN_ROOT="${EDEN_ROOT:-$HOME/.e166o}"     # SHORT eden.root so swtpm's 108-byte AF_UNIX control
                                           # socket path fits (a long path silently dead-TPMs).
SSH_TRIES="${SSH_TRIES:-40}"               # ssh-up poll attempts (x12s)
FILL_PCT="${FILL_PCT:-70}"                 # ext4-toofull: fill /persist to this used% so
                                           # used > maxFull(90%) x (P3-22G) => check insufficient.

# Workspace root (eden checkout): prep lives at <WS>/tests/update_eve_image/testdata,
# so three levels up. Used to pin eden.tests and the OVMF firmware in base_config.
WSROOT="$(cd "$(dirname "$0")/../../.." && pwd)"

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

image_dir() {
    # eden writes images under its configured eden.root, so that is the
    # authoritative location; EDEN_HOME / a workspace-relative dist/ are fallbacks.
    local root
    root=$(eden config get default --key eden.root 2>/dev/null | grep -vi 'level=' | tr -d '\r' | tail -1)
    [ -n "$root" ] && [ -d "$root/default-images/eve" ] && { echo "$root/default-images/eve"; return; }
    if [ -n "${EDEN_HOME:-}" ] && [ -d "$EDEN_HOME/default-images/eve" ]; then
        echo "$EDEN_HOME/default-images/eve"; return
    fi
    [ -d dist/default-images/eve ] && { echo dist/default-images/eve; return; }
    die "could not locate the eve image dir (checked eden.root='$root', EDEN_HOME, ./dist)"
}

certs_dir() {
    # Onboarding certs live at <eden.root>/default-certs; resolve the root like
    # image_dir so a non-default eden.root (e.g. ~/.e166o) is honored. A hardcoded
    # dist/default-certs misses them, so the certs survive and eden setup skips
    # grub.cfg -- which is what the grub-option ZFS legs rely on being rewritten.
    local root
    root=$(eden config get default --key eden.root 2>/dev/null | grep -vi 'level=' | tr -d '\r' | tail -1)
    [ -n "$root" ] && { echo "$root/default-certs"; return; }
    [ -n "${EDEN_HOME:-}" ] && { echo "$EDEN_HOME/default-certs"; return; }
    echo dist/default-certs
}

# The single-disk ZFS legs deliver eve_install_zfs_with_raid_level via grub.cfg,
# which eden only rewrites when the certs dir has no root-certificate.pem. A later
# leg that reuses the context therefore inherits that token and formats the boot
# P3 as ZFS -- silently mis-topologying the ext4 legs and colliding the twodisk-zfs
# `zpool create persist` against the accidental boot-disk pool. Non-ZFS-boot legs
# call this before do_setup so eden regenerates a token-free grub.cfg (P3 defaults
# to ext4). Mirrors the wipe the zfs legs use to force the token IN.
force_clean_grub() {
    eden stop || true
    rm -rf "$(image_dir)/live.img" "$(image_dir)/live.raw.qcow2" "$(certs_dir)"
    docker volume rm eden_adam_volume eden_redis_volume 2>/dev/null || true
}

# Fail loudly if the device came up on the wrong /persist filesystem, so a
# mis-topology is a legible error instead of a silent false pass. $1 = ext4|zfs.
assert_persist_type() {
    local got
    got=$(eve_ssh 'cat /run/eve.persist_type' | tr -d '\r' | tail -1)
    [ "$got" = "$1" ] || die "persist_type='$got' (expected $1 for $TOPOLOGY)"
}

base_config() {
    note "base eden config (TPM + accel + roam-proof + 64 GiB boot, tag $BRINGUP_TAG, root $EDEN_ROOT)"
    # Self-contained: SET every key the run depends on. `eden config add default`
    # leaves an already-existing context's keys intact, so any stale value silently
    # persists across legs (a prior leg's eve.disk=32768 is exactly what shrank the
    # boot disk to 32 GiB and made the shrink resize2fs fail). This mirrors the
    # validated bringup-grow.sh config block so prep does not depend on a
    # pre-configured context.
    if [ -f "$WSROOT/firmware/OVMF_CODE.fd" ] && [ -f "$WSROOT/firmware/OVMF_VARS.fd" ]; then
        eden config add default --devmodel ZedVirtual-4G \
            --eve-firmware "$WSROOT/firmware/OVMF_CODE.fd,$WSROOT/firmware/OVMF_VARS.fd"
    else
        eden config add default --devmodel ZedVirtual-4G
    fi
    eden config set default --key=eve.devmodel --value=ZedVirtual-4G
    eden config set default --key=eve.tpm      --value=true
    eden config set default --key=eve.accel    --value=true
    eden config set default --key=eve.tag      --value="$BRINGUP_TAG"
    # Pin the bringup hypervisor to kvm: the SMALL start is always -kvm (12.1.0 has
    # no -k image); a prior leg's kvm->k conversion leaves eve.hv=k, which makes
    # `eden setup` try to pull the nonexistent <tag>-k image. The escript's BaseOs
    # hop -- not eden setup -- is what moves the device to -k.
    eden config set default --key=eve.hv       --value=kvm
    # eve.disk sizes the BOOT live.img (via the EVE image's `live <MB>` generator),
    # not just the extra disks; 64 GiB single boot disk by default. Reset eve.disks
    # to a single disk (two-disk legs override to 1); without the reset a prior
    # two-disk leg's eve.disks=1 lingers and eden spuriously creates + write-locks
    # eve-disk-1.qcow2.
    eden config set default --key=eve.disk     --value="$BOOT_DISK_MB"
    eden config set default --key=eve.disks    --value=0
    # SHORT eden.root so swtpm's control socket path fits its 108-byte AF_UNIX cap.
    eden config set default --key=eden.root    --value="$EDEN_ROOT"
    # eden.tests must point at a real tests tree so escripts that run a NESTED
    # escript (e.g. cross_hv's Step-5 revert) can resolve {{EdenConfig "eden.tests"}}.
    eden config set default --key=eden.tests   --value="$WSROOT/tests"
    eden config set default --key=eve.hostfwd  --value='{"2222":"22","2223":"2223"}'
    # Roam-proof: host CLI -> containers via localhost (every eden container is
    # published there; adam's cert SAN includes 127.0.0.1), and EVE -> adam via the
    # QEMU slirp gateway (.2 of net=192.168.0.0/24). A laptop IP/WiFi/location change
    # then can't break onboarding, the redis-backed pod/volume queries, the eserver
    # upload, or the EVE-side controller path. See
    # feedback_eden_bringup_always_roamproof_slirp / feedback_eden_adam_ip_lan_sensitive.
    eden config set default --key=adam.ip         --value=127.0.0.1
    eden config set default --key=adam.redis.eden --value=127.0.0.1:6379
    eden config set default --key=eden.eserver.ip --value=127.0.0.1
    eden config set default --key=registry.ip     --value=127.0.0.1
    eden config set default --key=adam.eve-ip     --value=192.168.0.2
}

eve_ssh() { eden eve ssh -- "$1" 2>/dev/null | grep -v 'level=fatal'; }

wait_ssh() {
    note "waiting for EVE ssh (up to $((SSH_TRIES*12))s; a fresh EVE-kvm takes a few minutes to boot sshd)"
    local i
    for i in $(seq 1 "$SSH_TRIES"); do
        # stderr silenced: a not-yet-up sshd makes `eden eve ssh` log a scary
        # FATA/exit-255 that is NOT a failure of this script -- print our own
        # heartbeat instead so progress is visible during the boot wait.
        if timeout 8 eden eve ssh -- "true" >/dev/null 2>&1; then
            echo "  [wait_ssh] EVE ssh up (attempt $i/$SSH_TRIES)"; verify_console; return 0
        fi
        echo "  [wait_ssh] attempt $i/$SSH_TRIES: EVE not ssh-ready yet (still booting); retry in 12s"
        sleep 12
    done
    die "EVE ssh did not come up after $((SSH_TRIES*12))s"
}

# eden builds the live image by running the EVE image in a throwaway container and
# discards that container's exit code (RunDockerCommand), so an occasional racy
# generation surfaces only as a misleading "cannot copy ... live.raw.qcow2: no such
# file". Retry, wiping the half-built image dir and bumping verbosity, then confirm
# live.img exists. Extra args pass through to `eden setup` (e.g. --grub-options ...).
do_setup() {
    local attempt img="$EDEN_ROOT/default-images/eve/live.img"
    for attempt in 1 2 3; do
        if [ "$attempt" -eq 1 ]; then
            eden setup "$@" && [ -f "$img" ] && return 0
        else
            note "eden setup failed (try $((attempt-1))); wiping eve image dir, retrying with debug"
            rm -rf "$EDEN_ROOT/default-images/eve"
            eden -v debug setup "$@" && [ -f "$img" ] && return 0
        fi
        sleep 5
    done
    die "eden setup failed after retries (no $img)"
}

# swtpm's control socket is an AF_UNIX path capped at 108 bytes; a long eden.root
# overflows it and swtpm dies while eden boots a dead tpm-tis device anyway -- a
# silent ~12-min onboard timeout with no hint, and every vault/seal assert then
# meaningless. Verify swtpm is actually up right after start. See
# reference_eden_swtpm_enable_and_socket_path.
verify_swtpm() {
    local sock="$EDEN_ROOT/default-images/eve/swtpm/swtpm-sock"
    sleep 3
    if ! pgrep -x swtpm >/dev/null 2>&1 || [ ! -S "$sock" ]; then
        sed -n '1,5p' "$(dirname "$sock")/swtpm.log" 2>/dev/null
        die "swtpm not running / socket missing ($sock, len ${#sock}) — eve.tpm asserts would be meaningless"
    fi
    note "swtpm OK (pid $(pgrep -x swtpm | tr '\n' ' '))"
}

# Console-capture sanity: a mid-conversion stall is diagnosed from the serial log
# ($EDEN_ROOT/default-eve.log), so confirm the running qemu is writing a fresh log
# carrying our boot banner. A stale/empty log (content from an orphan qemu that
# outlived the reset) makes a post-mortem useless -- surface it now, as a WARN.
verify_console() {
    local clog="$EDEN_ROOT/default-eve.log" tag="${BRINGUP_TAG}-kvm-amd64"
    if [ ! -s "$clog" ]; then
        echo "  WARN: console log $clog missing/empty — a mid-run stall would be undiagnosable" >&2
    elif ! grep -aq "$tag" "$clog" 2>/dev/null; then
        echo "  WARN: console log $clog has no '$tag' boot banner (stale from an orphan qemu?)" >&2
    fi
}

# Enlarge a qcow2 boot disk by GROW_TAIL_GB and relocate the backup GPT so the
# free tail is usable (README_kvm_to_k_grow.md recipe). eden must be stopped.
add_free_tail() {
    local img="$1" fmt
    [ -f "$img" ] || die "boot image not found: $img"
    note "enlarging $img by +${GROW_TAIL_GB}G and relocating backup GPT (sudo-free)"
    cp -f "$img" "$img.prespare.bak"
    fmt=$(qemu-img info "$img" | sed -ne 's/^file format: //p')
    qemu-img resize "$img" "+${GROW_TAIL_GB}G" || die "boot resize failed"
    # `sgdisk -e` relocates the backup GPT to the new end + extends last-usable-LBA,
    # exposing the added space as a free tail. Run on a *file* (a qcow2 is round-
    # tripped through a sparse raw) — no qemu-nbd, no root; same trick as
    # grow_boot_no_tail.
    if [ "$fmt" = raw ]; then
        sgdisk -e "$img" || die "sgdisk -e failed"
        sgdisk -v "$img" || die "sgdisk -v failed"
    else
        local raw="$img.raw.tmp"; rm -f "$raw"
        qemu-img convert -f qcow2 -O raw "$img" "$raw" || die "qcow2->raw failed"
        sgdisk -e "$raw" || die "sgdisk -e on raw failed"
        sgdisk -v "$raw" || die "sgdisk -v on raw failed"
        qemu-img convert -f raw -O qcow2 "$raw" "$img" || die "raw->qcow2 failed"
        rm -f "$raw"
    fi
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

# Build an ext4 /persist on a whole SECOND disk (the two-disk-ext4 topology),
# entirely OFFLINE and sudo-free. sdb is still blank at this point (persist lived
# on the boot P3), so we rebuild its qcow2: a GPT with one partition named P3 (the
# EVE persist type GUID) filling the disk, ext4 inside it via `mkfs.ext4 -E offset`
# on a sparse raw image, then convert back to qcow2. storage-init's
# `findfs PARTLABEL=P3` then picks it up and mounts it ext4 on the next boot.
# Contrast with eve_make_persist_pool (ZFS): ext4 has no pool feature-flag
# constraint, so no in-guest step and no sudo/qemu-nbd are needed — any mkfs.ext4
# works. eden must be stopped. $1 = sdb qcow2 path (sized EVE_DISK_MB).
make_sdb_ext4_persist() {
    local img="$1" raw="$1.raw.tmp" start end off nblk
    [ -f "$img" ] || die "sdb qcow2 not found: $img"
    note "building ext4 /persist on sdb ($img): GPT P3 + mkfs.ext4, offline"
    rm -f "$raw"
    truncate -s "${EVE_DISK_MB}M" "$raw" || die "truncate sdb raw failed"
    sgdisk --largest-new=1 --typecode=1:5f24425a-2dfa-11e8-a270-7b663faccc2c \
        --change-name=1:P3 "$raw" >/dev/null || die "sgdisk sdb P3 failed"
    start=$(sgdisk -i 1 "$raw" | awk '/First sector:/{print $3}')
    end=$(sgdisk -i 1 "$raw" | awk '/Last sector:/{print $3}')
    { [ -n "$start" ] && [ -n "$end" ]; } || die "could not read sdb P3 sectors"
    off=$((start * 512)); nblk=$((end - start + 1))
    mkfs.ext4 -F -q -L PERSIST -E offset="$off" "$raw" "$((nblk / 2))k" || die "mkfs.ext4 sdb failed"
    qemu-img convert -f raw -O qcow2 "$raw" "$img" || die "sdb raw->qcow2 failed"
    rm -f "$raw"
}

# Offline-delete the boot disk's P3 (partition 9) so storage-init takes the no-P3
# -> `zpool import persist` path. eden must be stopped. $1 = boot qcow2.
delete_boot_p3() {
    # Superseded by the sudo-free delete_boot_p3_offline (raw round-trip); kept as a
    # thin alias so no run ever needs root.
    delete_boot_p3_offline "$@"
}

# Sudo-free variant of delete_boot_p3: delete the GPT partition named P3 via a
# qcow2<->raw round-trip (qemu-img convert is sparse-aware) instead of `sudo
# qemu-nbd`, so the two-disk-ext4 topology needs no root at all. Same GPT result;
# the freed tail becomes the grow target. eden must be stopped. $1 = boot image.
delete_boot_p3_offline() {
    local img="$1" fmt raw
    [ -f "$img" ] || die "boot image not found: $img"
    fmt=$(qemu-img info "$img" | sed -ne 's/^file format: //p')
    note "offline-deleting boot P3 from $img (sudo-free, fmt=$fmt)"
    _del_p3() {   # delete partition named P3 on block-or-file target $1; die on failure
        local t="$1" p3
        p3=$(sgdisk -p "$t" | awk '$NF=="P3"{print $1}')
        [ -n "$p3" ] || die "no GPT partition named P3 on boot disk"
        sgdisk -d "$p3" "$t" || die "sgdisk -d $p3 failed"
        sgdisk -p "$t" | awk '$NF=="P3"{f=1} END{exit !f}' && die "P3 still present after delete"
        return 0
    }
    if [ "$fmt" = raw ]; then
        _del_p3 "$img"
    else
        raw="$img.raw.tmp"; rm -f "$raw"
        qemu-img convert -f qcow2 -O raw "$img" "$raw" || die "qcow2->raw failed"
        _del_p3 "$raw"
        qemu-img convert -f raw -O qcow2 "$raw" "$img" || die "raw->qcow2 failed"
        rm -f "$raw"
    fi
    echo "boot P3 deleted"
}

wait_settled() {
    note "onboard + wait for boot (persist is created during boot, before sshd)"
    eden eve onboard || true
    wait_ssh
}

case "$TOPOLOGY" in

    ext4-shrink)
        # Single 64 GiB ext4 disk (from base_config), P3 fills it (no tail). The
        # conversion must SHRINK P3 (~63 GiB -> ~41 GiB) to grow ESP/IMGA/IMGB; on a
        # 64 GiB disk the ~41 GiB post-shrink persist holds the escript's 33 GiB fill,
        # which a 32 GiB disk (its ~10 GiB post-shrink persist) cannot -- that was the
        # resize2fs "New size smaller than minimum" failure.
        guard_no_other_eden
        base_config
        force_clean_grub
        note "eden setup (live.img, single 64 GiB ext4 disk, no tail)"
        do_setup; eden start; verify_swtpm; eden eve onboard; wait_ssh
        assert_persist_type ext4
        echo "READY: ext4-shrink. Run: EXPECT_DECISION=shrink ... kvm_to_k"
        ;;

    ext4-grow)
        # Single ext4 disk + >=22G free tail. 32 GiB boot (P3 fills it) + a 32 GiB
        # tail added after onboarding => 64 GiB total. live.img bringup, then enlarge.
        guard_no_other_eden
        base_config
        force_clean_grub
        eden config set default --key=eve.disk --value="$EVE_DISK_MB"  # 32 GiB boot; +32 GiB tail = 64
        note "eden setup (live.img, single ext4 disk)"
        do_setup; eden start; verify_swtpm
        wait_settled
        assert_persist_type ext4
        note "stopping eden to enlarge the boot disk"
        eden eve stop; sleep 5
        add_free_tail "$(image_dir)/live.img"
        note "resuming (eden start only — do NOT re-run eden setup; --force regenerates live.img)"
        eden start; verify_console
        echo "READY: ext4-grow. Run: EXPECT_DECISION=grow ... kvm_to_k"
        ;;

    twodisk-zfs)
        # VERIFIED 2026-06-22 (eden sandbox). Two disks: sda boot (P3 deleted =>
        # free tail), sdb ZFS /persist. EVE builds the pool; we delete boot P3.
        guard_no_other_eden
        base_config
        force_clean_grub
        eden config set default --key=eve.disks --value=1   # adds sdb
        eden config set default --key=eve.disk  --value="$EVE_DISK_MB"  # sizes sdb
        note "eden setup + start + onboard (sda boot ext4 P3, sdb blank)"
        do_setup; eden start; verify_swtpm; eden eve onboard
        wait_ssh
        # Boot P3 must be ext4 here: a stale ZFS token would put a `persist` pool on
        # the boot disk and collide with the sdb `zpool create` below.
        assert_persist_type ext4
        eve_make_persist_pool /dev/sdb
        note "stopping eden to delete the boot P3"
        eden eve stop; sleep 6
        delete_boot_p3 "$(image_dir)/live.img"
        note "restart — storage-init no-P3 branch imports persist from sdb"
        eden start
        wait_ssh
        note "verify"
        eve_ssh "echo persist_type=\$(cat /run/eve.persist_type); eve exec pillar sh -c \"zpool status persist; df -h /persist\""
        assert_persist_type zfs
        echo "READY: twodisk-zfs. Run: EXPECT_DECISION=grow DISK_TOPOLOGY=two-disk ... kvm_to_k"
        ;;

    twodisk-ext4)
        # VERIFIED 2026-07-07 (host eden; full kvm->k conversion PASSED via the
        # ext4mig-host-* scripts, of which this is the portable re-expression).
        # Two disks: sda boot (P3 deleted => free tail => resizer decides GROW),
        # sdb = an ext4 /persist on its OWN disk. Simpler than twodisk-zfs: the
        # persist is built OFFLINE with plain mkfs.ext4 (no in-guest pool, no
        # reserved/snapshots datasets), and kvm->k needs NO vault fs->zvol migration
        # (the ext4 vault carries over unchanged). Same grow code path as ZFS: the
        # resizer's `check --disk <bootdev>` finds no P3 on the boot disk =>
        # shrink N/A => grow off the free tail (pkg/storage-resizer evaluate/decide;
        # unit-tested as "multi-disk ext4, boot has free tail -> grow").
        # Ordering: the FIRST boot must create IMGB+P3 on the boot disk (storage-init
        # is_first_boot_virt_eve needs BOTH absent), so we boot once with persist on
        # the boot P3, THEN move persist to sdb + delete boot P3. Pre-seeding sdb's
        # P3 before first boot would suppress IMGB creation.
        guard_no_other_eden
        base_config
        force_clean_grub
        eden config set default --key=eve.disks --value=1   # adds sdb
        eden config set default --key=eve.disk  --value="$EVE_DISK_MB"  # sizes sdb
        note "eden setup"
        do_setup
        # Grow the boot disk before first boot so the P3 (created full) leaves a
        # >=22G free tail once deleted. grow_boot_no_tail is sudo-free (raw round-trip).
        grow_boot_no_tail "$(image_dir)/live.img" "$TWODISK_BOOT_GB"
        note "start + onboard (sda boot ext4 P3, sdb blank)"
        eden start; verify_swtpm; eden eve onboard
        wait_ssh
        note "stop; build sdb ext4 persist (offline) + delete boot P3"
        eden eve stop; sleep 6
        make_sdb_ext4_persist "$(image_dir)/eve-disk-1.qcow2"
        delete_boot_p3_offline "$(image_dir)/live.img"
        note "restart — storage-init finds P3 on sdb (ext4); boot disk has a free tail"
        eden start
        wait_ssh
        note "verify"
        eve_ssh "echo persist_type=\$(cat /run/eve.persist_type); eve exec pillar sh -c \"lsblk -o NAME,SIZE,FSTYPE,PARTLABEL /dev/sda /dev/sdb; df -h /persist\""
        assert_persist_type ext4
        echo "READY: twodisk-ext4. Run: EXPECT_DECISION=grow DISK_TOPOLOGY=two-disk ... kvm_to_k"
        ;;

    ext4-toofull)
        # Single 64 GiB ext4 disk (from base_config) so an EMPTY P3 could shrink to
        # free 22G; then /persist is filled to FILL_PCT% so the shrink CAN'T free
        # 22G within --max-full => check `insufficient` (reason=too-full). Refused
        # case. VERIFIED end-to-end on host eden 2026-07-01 (e2e testplan row C):
        # check `decision=insufficient persistType=ext4 shrinkApplicable=true
        # shrink.ok=false`, kvm->k declined with BaseOsStatus.Error
        # "conversion not possible: persist is too full to free the needed space".
        guard_no_other_eden
        base_config
        force_clean_grub
        note "eden setup (live.img, single 64 GiB ext4 disk)"
        do_setup
        eden start; verify_swtpm; eden eve onboard
        wait_ssh
        # A ZFS persist here would be the wrong topology (that is zfs-notail).
        assert_persist_type ext4
        fill_persist
        echo "READY: ext4-toofull. Run: REFUSE_REASON=too-full ... kvm_to_k_refused"
        ;;

    zfs-grow|zfs-notail)
        # SINGLE-disk ZFS /persist (persist == the boot disk's P3 as ZFS).
        # VERIFIED 2026-06-25. The token is delivered via grub.cfg, which eden only
        # (re)writes when CertsDir has no root-certificate.pem (setupConfigDir gate) —
        # so we MUST wipe the certs dir (certs_dir(), under eden.root) + adam/redis
        # volumes before setup,
        # else `eden setup --grub-options ...` silently skips grub.cfg and the token
        # never lands. live.img has no pre-baked P3, so storage-init creates it on
        # first boot and formats it ZFS because the token sets P3_FS_TYPE_DEFAULT=zfs
        # (storage-init.sh:113). Result: persist_type=zfs, zpool ONLINE on sda9.
        guard_no_other_eden
        base_config
        # zfs-grow needs a free tail (32 GiB boot + 32 GiB tail = 64); zfs-notail is
        # a full 64 GiB single disk (ZFS P3 fills it, no tail => refused).
        [ "$TOPOLOGY" = zfs-grow ] && eden config set default --key=eve.disk --value="$EVE_DISK_MB"
        note "wipe certs + adam/redis volumes so eden setup regenerates grub.cfg"
        eden stop || true
        rm -rf "$(image_dir)/live.img" "$(image_dir)/live.raw.qcow2" "$(certs_dir)"
        docker volume rm eden_adam_volume eden_redis_volume 2>/dev/null || true
        note "eden setup with the grub-option (canonical bare token, set_global form)"
        # $dom0_extra_args is a grub variable expanded at boot; it must reach grub
        # literally, so keep the single quotes and do not let the shell expand it.
        # shellcheck disable=SC2016
        do_setup --grub-options 'set_global dom0_extra_args "$dom0_extra_args eve_install_zfs_with_raid_level "'
        [ -f "$(certs_dir)/grub.cfg" ] || die "grub.cfg not written — certs were not wiped?"
        eden start; verify_swtpm; eden eve onboard
        wait_ssh
        note "confirm the cmdline took effect"
        eve_ssh "echo cmdline=\$(cat /proc/cmdline); echo persist_type=\$(cat /run/eve.persist_type)"
        # Expect: cmdline contains eve_install_zfs_with_raid_level AND persist_type=zfs.
        if [ "$TOPOLOGY" = zfs-grow ]; then
            note "zfs-grow: add a >=22G free tail after the ZFS P3"
            eden eve stop; sleep 5
            add_free_tail "$(image_dir)/live.img"   # ZFS partition is not auto-grown -> leaves a tail
            eden start; verify_console
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
