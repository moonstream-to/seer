# seer

`seer` is blockchain-adjacent tooling for crawling data and performing smart contract interactions. It has three main functions: 
1. Generating a smart contract interface
2. Crawling and indexing blockchain data
3. Checking for corrupted data in database/storage of crawled blocks

### Prerequisites 

[Go](https://go.dev/) (version >= 1.21)

To install `seer`:
```bash
go install https://github.com/G7DAO/seer@latest
```
## 1. Generating a smart contract interface

`seer` simplifies interactions with deployed smart contracts by generating a golang interface. It lets you access contract functions (like balance checks or transactions) through a command-line interface (CLI), instead of using web-based blockchain explorers like Etherscan.

`seer` also lets you use pre-saved CLI commands from .md files for faster interaction with smart contracts.
It helps quickly develop auto generated scripts for interacting with contracts, saving you from writing complex custom scripts every time you need to interact with the blockchain (e.g., deploying, checking balances, transferring funds).

It doesn’t come with pre-written scripts. You need to generate your own scripts using `seer`, following examples in this README.

You can use [`Makefile`](./Makefile) as an example. It is recommended to have a makefile that contains the `seer evm generate` commands for each of your smart contracts.

## Build

You can use `make` to build `seer`. From the root of this project, run:

```bash
make
```

If you would like to force a build from scratch:

```bash
make rebuild
```

To see a list of available `make` targets, please look at the [`Makefile`](./Makefile).

## Running seer

### Go bindings for Starknet contracts

To generate the Go bindings to a Starknet contract from its ABI, run:

```bash
seer starknet generate --abi $ABI_FILE --package $GO_PACKAGE_NAME
```

1. `$ABI_FILE` should be the path to the JSON file containing the Starknet contract ABI.
2. `$GO_PACKAGE_NAME` should be the name of the Go package that the generated code will belong to. If specified, the line `package $GO_PACKAGE_NAME` will be emitted at the top of the generated code. If not specified, no such line is emitted.

You can also pipe the ABI JSON into this command rather than specifying the `--abi` argument. For example:

```bash
jq . $ABI_FILE | seer starknet generate --package $GO_PACKAGE_NAME
```

### Go bindings for Ethereum Virtual Machine (EVM) contracts

To generate the Go bindings to an EVM contract, run:

```bash
seer evm generate \
    --abi $ABI_FILE \
    --bytecode $BIN_FILE \
    --cli \
    --package $GO_PACKAGE_NAME \
    --struct $GO_STRUCT_NAME
```

1. `$ABI_FILE` should be the path to a JSON file containing the contract's ABI.
2. `$BIN_FILE` should be a path to the file containing the compiled contract bytecode. If the `--bytecode` is not provided, the bindings are generated with no deployment method.
3. `$GO_PACKAGE_NAME` should be the name of the Go package that the generated code will fall under.
4. `$GO_STRUCT_NAME` should be the name of the struct that you would like to represent an instance of the contract with the given ABI.

If you want to write the output to a file, you can use the `--output` argument to do so. Or shell redirections.

#### Example: how to use a go binding as a CLI

Generating an ERC20 go binding:
1. Initialize it as a go package. It’s important to have the main package included (`--includemain flag`) to create a standalone CLI for a contract. This will generate a go .mod file.
```bash
go mod init ERC20
```

2. You can now install all the dependencies. To install every module from our package and generate a go.sum file: 
```bash
go mod tidy
```

3. Now to generate a CLI file (ERC20 file without an extension) you can use 
```bash
go build .
```
 
4. To see all the go bindings functions that are analogous to the contract and can be called directly from the CLI use
```bash
 ./ERC20 -h
```

#### Example: `OwnableERC721`

The code in [`examples/ownable-erc-721`](./examples/ownable-erc-721/OwnableERC721.go) was generated from the project root directory using:

```bash
seer evm generate \
    --abi fixtures/OwnableERC721.json \
    --bytecode fixtures/OwnableERC721.bin \
    --cli \
    --package main \
    --struct OwnableERC721 \
    --output examples/ownable-erc-721/OwnableERC721.go
```

To run this code, first build it:

```bash
go build ./examples/ownable-erc-721
```

This will create an executable file called `ownable-erc-721` (on Windows, you may want to rename it to `ownable-erc-721.exe` for convenience).

Try running it:

```bash
$ ./ownable-erc-721 -h

Interact with the OwnableERC721 contract

Usage:
  ownable-erc-721 [flags]
  ownable-erc-721 [command]

Commands which deploy contracts
  deploy               Deploy a new OwnableERC721 contract
  verify               Verify a new OwnableERC721 contract

Commands which view contract state
  balance-of           Call the BalanceOf view method on a OwnableERC721 contract
  get-approved         Call the GetApproved view method on a OwnableERC721 contract
  is-approved-for-all  Call the IsApprovedForAll view method on a OwnableERC721 contract
  name                 Call the Name view method on a OwnableERC721 contract
  owner                Call the Owner view method on a OwnableERC721 contract
  owner-of             Call the OwnerOf view method on a OwnableERC721 contract
  supports-interface   Call the SupportsInterface view method on a OwnableERC721 contract
  symbol               Call the Symbol view method on a OwnableERC721 contract
  token-uri            Call the TokenURI view method on a OwnableERC721 contract

Commands which submit transactions
  approve              Execute the Approve method on a OwnableERC721 contract
  mint                 Execute the Mint method on a OwnableERC721 contract
  renounce-ownership   Execute the RenounceOwnership method on a OwnableERC721 contract
  safe-transfer-from   Execute the SafeTransferFrom method on a OwnableERC721 contract
  safe-transfer-from-0 Execute the SafeTransferFrom0 method on a OwnableERC721 contract
  set-approval-for-all Execute the SetApprovalForAll method on a OwnableERC721 contract
  transfer-from        Execute the TransferFrom method on a OwnableERC721 contract
  transfer-ownership   Execute the TransferOwnership method on a OwnableERC721 contract

Additional Commands:
  completion           Generate the autocompletion script for the specified shell
  help                 Help about any command

Flags:
  -h, --help   help for ownable-erc-721

Use "ownable-erc-721 [command] --help" for more information about a command.
```

```bash
$ ./ownable-erc-721 approve -h
Execute the Approve method on a OwnableERC721 contract

Usage:
  ownable-erc-721 approve [flags]

Flags:
      --contract string                   Address of the contract to interact with
      --gas-limit uint                    Gas limit for the transaction
      --gas-price string                  Gas price to use for the transaction
  -h, --help                              help for approve
      --keyfile string                    Path to the keystore file to use for the transaction
      --max-fee-per-gas string            Maximum fee per gas to use for the (EIP-1559) transaction
      --max-priority-fee-per-gas string   Maximum priority fee per gas to use for the (EIP-1559) transaction
      --nonce string                      Nonce to use for the transaction
      --password string                   Password to use to unlock the keystore (if not specified, you will be prompted for the password when the command executes)
      --rpc string                        URL of the JSONRPC API to use
      --safe string                       Address of the Safe contract
      --safe-api string                   Safe API for the Safe Transaction Service (optional)
      --safe-operation uint8              Safe operation type: 0 (Call) or 1 (DelegateCall)
      --simulate                          Simulate the transaction without sending it
      --timeout uint                      Timeout (in seconds) for interactions with the JSONRPC API (default 60)
      --to-0 string                       to-0 argument (common.Address)
      --token-id string                   token-id argument
      --value string                      Value to send with the transaction
```

## 2. Crawling and indexing blockchain data

This part of `seer` is responsible for crawling raw blocks,tx_calls and events from the blockchain.

List of supported blockchains:

-   arbitrum_one
-   arbitrum_sepolia
-   ethereum
-   game7_orbit_arbitrum_sepolia
-   mantle
-   mantle_sepolia
-   polygon
-   xai
-   xai_sepolia

## Build

You can use `make` to build `crawler`. From the root of this project, run:

```bash
make
```

Or build with go tools:

```bash
go build -o seer .
```

Or use dev script:

```bash
./dev.sh --help
```

Set environment variables:

```bash
export MOONSTREAM_DB_V3_INDEXES_URI="driver://user:pass@localhost/dbname"
```

## Generate crawler {chain} interface

note: You need add the chain endpoint it will fetch the data from endpoints.

Blockchain structure:

```bash
├── blockchain
│   ├── polygon
│   │   ├── blocks_transactions_polygon.proto
│   │   ├── blocks_transactions_polygon.pb.go
│   │   ├── polygon.go
```

## Regenerate proto interface

1. Generate code with proto compiler, docs: https://protobuf.dev/reference/go/go-generated/

```bash
protoc --go_out=. --go_opt=paths=source_relative \
    blockchain/ethereum/ethereum_index_types.proto
```

2. Rename generated file similar to chain package.
3. Generate interface with `seer`, if chain is L2, specify flag `--side-chain`:

```bash
./seer blockchain generate -n ethereum
```

Or use bash script to do all job. But be careful it by default generates interfaces for L1 chains with additional fields, for side chains this script requires modification:

```bash
./prepare_blockchains.sh
```

## Run crawler

Before running the crawler, you need initialize the database with the following command:

```bash
./seer crawler --chain polygon --start-block 53922484 --force
```
```
Usage:
  seer crawler [flags]

Flags:
      --base-dir string         The base directory to store the crawled data
      --batch-size int          Dynamically changed maximum number of blocks to crawl in each batch (default 10)
      --chain string            The blockchain to crawl (default "ethereum")
      --confirmations int       The number of confirmations to consider for block finality (default 10)
      --final-block int         The block number to end crawling at
  -h, --help                    help for crawler
      --proto-size-limit uint   Proto file size limit in Mb (default 25)
      --proto-time-limit int    Proto time limit in seconds (default 300)
      --retry-multiplier int    Multiply wait time to get max waiting time before fetch new block (default 24)
      --retry-wait int          The wait time for the crawler in milliseconds before it try to fetch new block (default 5000)
      --start-block int         The block number to start crawling from (default: fetch from database, if it is empty, run from latestBlockNumber minus shift)
      --threads int             Number of go-routines for concurrent crawling (default 1)
      --timeout int             The timeout for the crawler in seconds (default 30)
```

Crawler command:

By setup --chain [ethreum, polygon,etc..] all available chains defined in **blockchain** folder.
`Proto files definitions and client interfaces is required` (command `seer blockchain generate -n $BLOCKCHAIN --side-chain` for generate them)


 You will run indexing of raw blockchain data 
depends on env var **SEER_CRAWLER_STORAGE_TYPE** to `["gcp-storage","filesystem"]` you will set where that data will be save on your local storage on in GCS.
Option "gcp-storage" required google credential setup on system or by variable **MOONSTREAM_STORAGE_GCP_SERVICE_ACCOUNT_CREDS_PATH**.

SET **SEER_CRAWLER_STORAGE_TYPE** to "filesystem"

run of crawler 
```
seer crawler ---cahin polygon
```

That will start create batch of blocks ranges indexed raw blocks/transactions/events data from blockchain.

```
dev
   /data
      /polygon
             /59193672-59193737:
                 - data.proto
             /59193738-59193781
                 - data.proto
```

All data in files in proto message format defined in `/seer/blockchain/<chain_name>/<chain_name>.proto`


## 3. Checking for corrupted data in database/storage of crawled blocks

Find first and last blocks indexed in database and verify it's batch at storage:

```bash
./seer inspector db --chain polygon --storage-verify
```

You can look into the crawled data directly using inspector

`seer inspector`

Inspect storage and database consistency

Usage:
  seer inspector [command]

Available Commands:
  - **db**          Inspect database consistency
  - **read**        Read and decode indexed proto data from storage
  - **storage**     Inspect filesystem, gcp-storage, aws-bucket consistency


`seer inspector read`

For see batch of 59193672-59193737 data in console run

```
seer inspector read --chain polygon --batch 59193738-59193781
```
Output
```
All blocks data of that batch
```

`seer inspector db`

`seer inspector storage`
