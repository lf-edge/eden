# LIO in eve

Инструкция по ручной натсройке vHost в EVE на примере target fileio

## Example

First, let's connect the necessary kernel modules:

```console
modprobe target_core_mod
modprobe target_core_file
modprobe vhost_scsi
modprobe vhost_net
```

### Create fileIO target

Start FILEIO subsystem plugin objects:

> Here the name for fileio is `fileio`.
> Each image requires a new directory to be created fileio_0. Example: fileio_0 fileio_1 fileio_2 for 3 image or file.

```console
mkdir -p /sys/kernel/config/target/core/fileio_0/fileio
```

Create fileio device:

```console
echo "fd_dev_name=<Device>,fd_dev_size=<Device size in byte>,fd_buffered_io=1" > /sys/kernel/config/target/core/fileio_0/fileio/control
```

Params:

- **fd_dev_name** The path to the file/image. Example: fd_dev_size=/persist/image.qcow2
- **fd_dev_size** Size of the specified file/disk/image in bytes. Example: fd_dev_size=1073741824 (1G). If we set up a block_device (Not a file), we don't have to set the size.
- **fd_buffered_io** True if write-back, False if write-through (write cache disabled). Example: fd_buffered_io=1
- **fd_async_io**  True if asynchronous I/O. (Set 1 for true & set 0 for false (default))

Set block_size:

```console
echo 4096 > /sys/kernel/config/target/core/fileio_0/fileio/attrib/block_size
```

Set T10 WWN Unit Serial number:

>We need to generate this wwn first

```console
echo "<T10 WWN Unit Serial>" > /sys/kernel/config/target/core/fileio_0/fileio/wwn/vpd_unit_serial
```

Set udev_path:

```console
echo -n '<Device>' >/sys/kernel/config/target/core/fileio_0/fileio/udev_path
```

Enable fileio device:

```console
echo 1 > /sys/kernel/config/target/core/fileio_0/fileio/enable
```

### Create vHost fabric

**Several things are important to Vhost:**

- nexus The TPGTs have a 'nexus' attribute that contains the local initiator serial unit. This attribute must be set before being able to create any LUNs.
- acls The target's TPGTs support explicit initiator ACLs.
- tpgts The target fabric module is using iSCSI-style target portal group tags.

> The wwn type for vhost must be 'naa.'

**Created target and tpgt:**

```console
mkdir -p /sys/kernel/config/target/vhost/<nna.1111111111111111>/tpgt_1/lun/lun_0
```

> If you need to create another vhost, repeat the same command, remembering to change wwn.

**Set (SCSI %d:%d:%d:%d) is in h:c:t:l format:**

```console
echo -n 'scsi_host_id=1,scsi_channel_id=0,scsi_target_id=0,scsi_lun_id=0' > /sys/kernel/config/target/core/fileio_0/fileio/control
```

**Enable device (if not enabled):**

```console
echo -n 1 > /sys/kernel/config/target/core/fileio_0/fileio/enable
```

**Set nexus:**

```console
echo -n <nna.222222222222222> > /sys/kernel/config/target/vhost/<nna.1111111111111111>/tpgt_1/nexus
```

**Create a link between LUN and vhost:**

```console
cd /sys/kernel/config/target/vhost/<nna.1111111111111111>/tpgt_1/lun/lun_0
ln -s ../../../../../core/fileio_0/fileio/ .
```

### QEMU

In the next step, we need to adjust the configuration in QEMU for the virtual machine image to fit vHost.

Add parameter for QEMU:

```console
-device vhost-scsi-pci,wwpn=<nna.1111111111111111>,bus=pci.0,addr=0x5
```

### Tree /sys/kernel/config/target/ (for example)

```code
├── core
│   ├── alua
│   │   └── lu_gps
│   │       └── default_lu_gp
│   │           ├── lu_gp_id
│   │           └── members
│   └── fileio_0
│       ├── fileio
│       │   ├── action
│       │   ├── alias
│       │   ├── alua
│       │   │   └── default_tg_pt_gp
│       │   │       ├── alua_access_state
│       │   │       ├── alua_access_status
│       │   │       ├── alua_access_type
│       │   │       ├── alua_support_active_nonoptimized
│       │   │       ├── alua_support_active_optimized
│       │   │       ├── alua_support_lba_dependent
│       │   │       ├── alua_support_offline
│       │   │       ├── alua_support_standby
│       │   │       ├── alua_support_transitioning
│       │   │       ├── alua_support_unavailable
│       │   │       ├── alua_write_metadata
│       │   │       ├── implicit_trans_secs
│       │   │       ├── members
│       │   │       ├── nonop_delay_msecs
│       │   │       ├── preferred
│       │   │       ├── tg_pt_gp_id
│       │   │       └── trans_delay_msecs
│       │   ├── alua_lu_gp
│       │   ├── attrib
│       │   │   ├── alua_support
│       │   │   ├── block_size
│       │   │   ├── emulate_3pc
│       │   │   ├── emulate_caw
│       │   │   ├── emulate_dpo
│       │   │   ├── emulate_fua_read
│       │   │   ├── emulate_fua_write
│       │   │   ├── emulate_model_alias
│       │   │   ├── emulate_pr
│       │   │   ├── emulate_rest_reord
│       │   │   ├── emulate_tas
│       │   │   ├── emulate_tpu
│       │   │   ├── emulate_tpws
│       │   │   ├── emulate_ua_intlck_ctrl
│       │   │   ├── emulate_write_cache
│       │   │   ├── enforce_pr_isids
│       │   │   ├── force_pr_aptpl
│       │   │   ├── hw_block_size
│       │   │   ├── hw_max_sectors
│       │   │   ├── hw_pi_prot_type
│       │   │   ├── hw_queue_depth
│       │   │   ├── is_nonrot
│       │   │   ├── max_unmap_block_desc_count
│       │   │   ├── max_unmap_lba_count
│       │   │   ├── max_write_same_len
│       │   │   ├── optimal_sectors
│       │   │   ├── pgr_support
│       │   │   ├── pi_prot_format
│       │   │   ├── pi_prot_type
│       │   │   ├── pi_prot_verify
│       │   │   ├── queue_depth
│       │   │   ├── unmap_granularity
│       │   │   ├── unmap_granularity_alignment
│       │   │   └── unmap_zeroes_data
│       │   ├── control
│       │   ├── enable
│       │   ├── info
│       │   ├── lba_map
│       │   ├── pr
│       │   │   ├── res_aptpl_active
│       │   │   ├── res_aptpl_metadata
│       │   │   ├── res_holder
│       │   │   ├── res_pr_all_tgt_pts
│       │   │   ├── res_pr_generation
│       │   │   ├── res_pr_holder_tg_port
│       │   │   ├── res_pr_registered_i_pts
│       │   │   ├── res_pr_type
│       │   │   └── res_type
│       │   ├── statistics
│       │   │   ├── scsi_dev
│       │   │   │   ├── indx
│       │   │   │   ├── inst
│       │   │   │   ├── ports
│       │   │   │   └── role
│       │   │   ├── scsi_lu
│       │   │   │   ├── creation_time
│       │   │   │   ├── dev
│       │   │   │   ├── dev_type
│       │   │   │   ├── full_stat
│       │   │   │   ├── hs_num_cmds
│       │   │   │   ├── indx
│       │   │   │   ├── inst
│       │   │   │   ├── lu_name
│       │   │   │   ├── lun
│       │   │   │   ├── num_cmds
│       │   │   │   ├── prod
│       │   │   │   ├── read_mbytes
│       │   │   │   ├── resets
│       │   │   │   ├── rev
│       │   │   │   ├── state_bit
│       │   │   │   ├── status
│       │   │   │   ├── vend
│       │   │   │   └── write_mbytes
│       │   │   └── scsi_tgt_dev
│       │   │       ├── aborts_complete
│       │   │       ├── aborts_no_task
│       │   │       ├── indx
│       │   │       ├── inst
│       │   │       ├── non_access_lus
│       │   │       ├── num_lus
│       │   │       ├── resets
│       │   │       └── status
│       │   ├── udev_path
│       │   └── wwn
│       │       ├── product_id
│       │       ├── revision
│       │       ├── vendor_id
│       │       ├── vpd_assoc_logical_unit
│       │       ├── vpd_assoc_scsi_target_device
│       │       ├── vpd_assoc_target_port
│       │       ├── vpd_protocol_identifier
│       │       └── vpd_unit_serial
│       ├── hba_info
│       └── hba_mode
├── dbroot
├── version
└── vhost
    ├── discovery_auth
    ├── naa.60014059811d880b
    │   ├── fabric_statistics
    │   └── tpgt_1
    │       ├── acls
    │       ├── attrib
    │       │   └── fabric_prot_type
    │       ├── auth
    │       ├── lun
    │       │   └── lun_0
    │       │       ├── alua_tg_pt_gp
    │       │       ├── alua_tg_pt_offline
    │       │       ├── alua_tg_pt_status
    │       │       ├── alua_tg_pt_write_md
    │       │       ├── fileio -> ../../../../../../target/core/fileio_0/fileio
    │       │       └── statistics
    │       │           ├── scsi_port
    │       │           │   ├── busy_count
    │       │           │   ├── dev
    │       │           │   ├── indx
    │       │           │   ├── inst
    │       │           │   └── role
    │       │           ├── scsi_tgt_port
    │       │           │   ├── dev
    │       │           │   ├── hs_in_cmds
    │       │           │   ├── in_cmds
    │       │           │   ├── indx
    │       │           │   ├── inst
    │       │           │   ├── name
    │       │           │   ├── port_index
    │       │           │   ├── read_mbytes
    │       │           │   └── write_mbytes
    │       │           └── scsi_transport
    │       │               ├── dev_name
    │       │               ├── device
    │       │               ├── indx
    │       │               ├── inst
    │       │               └── proto_id
    │       ├── nexus
    │       ├── np
    │       └── param
    └── version
```
