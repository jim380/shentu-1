## certikcli query supply total

Query the total supply of coins of the chain

### Synopsis

Query total supply of coins that are held by accounts in the
			chain.

Example:
$ certikcli query supply total

To query for the total supply of a specific coin denomination use:
$ certikcli query supply total stake

```
certikcli query supply total [denom] [flags]
```

### Options

```
      --height int    Use a specific height to query state at (this can error if the node is pruning state)
  -h, --help          help for total
      --indent        Add indent to JSON response
      --ledger        Use a connected Ledger device
      --node string   <host>:<port> to Tendermint RPC interface for this chain (default "tcp://localhost:26657")
      --trust-node    Trust connected full node (don't verify proofs for responses)
```

### Options inherited from parent commands

```
      --chain-id string   Chain ID of tendermint node
  -e, --encoding string   Binary encoding (hex|b64|btc) (default "hex")
      --home string       directory for config and data (default "~/.certikcli")
  -o, --output string     Output format (text|json) (default "text")
      --trace             print out full stack trace on errors
```

### SEE ALSO

* [certikcli query supply](certikcli_query_supply.md)	 - Querying commands for the supply module


