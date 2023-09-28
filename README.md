# Go-Tezos-Keygen

## Usage
```
Usage of ./go-tezos-keygen:
  -a string
    	Address (default ":3000")
  -d string
    	Database
  -l string
    	Level (default "info")
  -n string
    	Networks configuration file
  -seed
    	Generate seed and exit
```

Example:
```sh
./go-tezos-keygen -d db.db -n networks.yaml -l debug
```

## Networks file
The networks configuration file uses YAML syntax. Example:
```yaml
testnet:
  url: https://nairobinet.ecadinfra.com
  chain-id: NetXyuzvDo2Ugzb
  seed: f7353829d316c20922f8ff2ed696090801d9c775977df6423cd68a737c628b844b13c951de7cb7cd01cd62430edeefbc219885b388f06cb5d1e496f63bc9c0d5
  private-key: edsk2mgqWz5tUQQPK2LCg4Ae2G9bdd8RGzJP9oR3S7cKgASndbnRjE
  min-balance: 100000
  amount: 2000000
  ops-per-group: 5
  lease-time: 1m
  buffer-length: 10
  buffer-threshold: 0
  rpc-timeout: 2m
```

### Network options
#### `url`
Tezos node RPC URL.

#### `chain-id`
The proper Base58 chain id.

#### `seed`
512 byte seed from which all keys are being derived using SLIP-10 algorithm. Use `seed-file` to read the hex encoded seed from an external file. The seed can also be specified using an environment variable `NET_SEED`
where `NET` prefix is the network name in uppercase.

#### `private-key`
The Base58 encoded funding wallet key. Use `private-key-file` to read the Base58 encoded key from an external file. The key can also be specified using an environment variable `NET_PRIVATE_KEY` where `NET` prefix is the network name in uppercase.

#### `min-balance`
Minimal residual balance. The 'ephemeral' key will be discarded if its balance is below this value.

#### `amount`
The funding amount.

#### `ops-per-group`
The maximum number of transactions signed and injected as a single group.

#### `lease-time`
The duration after which the ephemeral key gets recycled.

#### `buffer-length`
The number of pre-funded keys in the queue.

#### `buffer-threshold`
Refill the queue when its length hits this value.

#### `rpc-timeout`
Tezos RPC timeout.

## Environment variables
#### `KEYGEN_NETWORKS`
Can be used as an alternative to `-n` command line option

### `KEYGEN_NETWORKS_DATA`
Contains network configuration in YAML format. Overrides both `-l` option and `KEYGEN_NETWORKS` variable.

#### `KEYGEN_DB`
Can be used as an alternative to `-d` command line option

### `*_PRIVATE_KEY`
See `private-key` above

### `*_SEED`
See `seed` above