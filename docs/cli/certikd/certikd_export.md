## certikd export

Export state to JSON

### Synopsis

Export state to JSON

```
certikd export [flags]
```

### Options

```
      --for-zero-height          Export state to start at height zero (perform preproccessing)
      --height int               Export state from a particular height (-1 means latest height) (default -1)
  -h, --help                     help for export
      --jail-whitelist strings   List of validators to not jail state export
```

### Options inherited from parent commands

```
      --home string        directory for config and data (default "~/.certikd")
      --log_level string   Log level (default "main:info,state:info,*:error")
      --trace              print out full stack trace on errors
```

### SEE ALSO

* [certikd](certikd.md)	 - CertiK App Daemon (server)


