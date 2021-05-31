#!/usr/bin/expect -f
set key_file [lindex $argv 0]
set passphrase [lindex $argv 1]
spawn ssh-add $key_file
expect "Enter passphrase for $key_file:"
send "$passphrase\n";
expect "Identity added: $key_file"
interact