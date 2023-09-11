package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/config"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/jurteam/tools/internal/csv"
)

var (
	flagEndpoint string
	flagDebug    bool
	flagFilename string
)

var ENDPOINTS = map[string]string{
	"simplystaking": "wss://jur-archive-mainnet-1.simplystaking.xyz/VX68C07AR4K2/ws",
	"iceberg":       "wss://jur-mainnet-archive-rpc-1.icebergnodes.io",
}

var (
	debugLogger *log.Logger
	signOpts    types.SignatureOptions
)

func init() {
	flag.StringVar(&flagEndpoint, "endpoint", "simplystaking", "RPC endpoint, choices: 'simplystaking', 'iceberg'")
	flag.StringVar(&flagFilename, "filename", "batch.csv", "batch transaction filename")
	flag.BoolVar(&flagDebug, "debug", true, "debug mode")

	debugLogger = log.New(&dummyWriter{}, "", 0)
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("batchxfer: ")
	log.SetOutput(os.Stderr)
	flag.CommandLine.SetOutput(os.Stderr)
	flag.Parse()

	if flagDebug {
		debugLogger = log.New(os.Stderr, "", 0)
		debugLogger.SetPrefix("DEBUG: ")
	}

	fp, err := os.Open(flagFilename)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			log.Fatalf("couildn't find %s: %v", flagFilename, err)
		case os.IsPermission(err):
			log.Fatalf("couildn't open %s: %v", flagFilename, err)
		default:
			log.Fatalf("an error occurred: %v", err)
		}
	}

	defer fp.Close()
	reader := csv.NewReader(fp)

	cfg := config.Default()
	cfg.RPCURL = ENDPOINTS[flagEndpoint]

	_, meta, err := setupConnection(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	senderAddr, err := reader.Sender()
	if err != nil {
		log.Fatal(err)
	}
	debug("senderAddress:", senderAddr)

	calls := make([]types.Call, 0)
	for !reader.EOF() {
		record, err := reader.Read()
		if err != nil && err == io.EOF {
			log.Println("EOF")
			break
		}

		if err != nil {
			log.Printf("Record: error encountered while parsing the CSV file: %v", err)
			continue
		}

		debug("recipient:", record.Address(), "amount:", record.Amount())

		c, err := types.NewCall(meta, "Balances.transfer", record.Address(), types.NewUCompactFromUInt(record.Amount()))
		if err != nil {
			log.Printf("error in creating a new call: %v", err)
			continue
		}

		calls = append(calls, c)
	}

	// Create the batch call
	batchCall, err := types.NewCall(meta, "utility.batch", calls)
	if err != nil {
		log.Fatal(err)
	}

	ext := types.NewExtrinsic(batchCall)
	fmt.Printf("%v", ext)
}

func setupConnection(cfg *config.Config) (*gsrpc.SubstrateAPI, *types.Metadata, error) {
	api, err := gsrpc.NewSubstrateAPI(cfg.RPCURL)
	if err != nil {
		return nil, nil, err
	}

	chain, err := api.RPC.System.Chain()
	if err != nil {
		return nil, nil, err
	}

	nodeName, err := api.RPC.System.Name()
	if err != nil {
		return nil, nil, err
	}

	nodeVersion, err := api.RPC.System.Version()
	if err != nil {
		return nil, nil, err
	}

	debug(fmt.Sprintf("You are connected to chain %v using %v v%v\n", chain, nodeName, nodeVersion))

	latestBlock, err := api.RPC.Chain.GetBlockLatest()
	if err != nil {
		log.Fatal(err)
	}
	debug("Latest block height:", latestBlock.Block.Header.Number)

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, nil, err
	}

	debug("Chain's metadata:", meta.Version)

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		log.Fatal(err)
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		log.Fatal(err)
	}

	signOpts = types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	return api, meta, nil
}

func debug(v ...any) {
	debugLogger.Println(v...)
}

type dummyWriter struct{}

func (*dummyWriter) Write([]byte) (int, error) { return 0, nil }
