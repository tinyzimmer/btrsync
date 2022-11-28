## btrsync send

Send a snapshot

```
btrsync send [flags] <subvolumes>...
```

### Options

```
  -z, --compressed      send compressed data
  -f, --force           force source to be readonly if it already isn't
  -h, --help            help for send
  -o, --output string   send to encoded file
```

### Options inherited from parent commands

```
  -c, --config string   config file
  -v, --verbose count   verbosity level (can be used multiple times)
```

### SEE ALSO

* [btrsync](btrsync.md)	 - A tool for syncing btrfs subvolumes and snapshots

###### Auto generated by spf13/cobra on 28-Nov-2022