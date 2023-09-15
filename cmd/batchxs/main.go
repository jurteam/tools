package main

import (
	"encoding/csv"

	//	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/config"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

var (
	flagEndpoint   string
	flagDebug      bool
	flagFilename   string
	flagSimulate   bool
	flagSkipErrors bool

	debugLogger *log.Logger
	rpcAddr     string
	signOpts    types.SignatureOptions
	inFile      io.ReadCloser
)

const (
	rpcSimplyStaking string = "wss://jur-archive-mainnet-1.simplystaking.xyz/VX68C07AR4K2/ws"
	rpcIceberg       string = "wss://jur-mainnet-archive-rpc-1.icebergnodes.io"
	rpcLocal         string = "wss://localhost"
)

func init() {
	flag.StringVar(&flagEndpoint, "endpoint", "localhost",
		"RPC endpoint, choices: 'simplystaking', 'iceberg'")
	flag.StringVar(&flagFilename, "filename", "-", "Batch transaction filename")
	flag.BoolVar(&flagSimulate, "s", false, "Simulate operations with actually executing them.")
	flag.BoolVar(&flagDebug, "D", true,
		"Run in simulate mode (it implies the -s flag) and print additional debug informaton to stderr. "+
			"Export the malformed records to a rejected_TIMESTAMP.csv file.")
	flag.BoolVar(&flagSkipErrors, "f", false,
		"Create and execut a batch of the valid records found in the input file. "+
			"Export malformed records to skipped_TIMESTAMP.csv")

	debugLogger = log.New(&dummyWriter{}, "", 0)
}

func handleFlags() {
	flagDebug = flagSimulate || flagDebug

	switch flagEndpoint {
	case "simply":
		rpcAddr = rpcSimplyStaking
	case "iceberg":
		rpcAddr = rpcIceberg
	case "local", "localhost":
		rpcAddr = "ws://127.0.0.1:9944/"
	default:
		log.Fatalf("invalid endooint: %q", flagEndpoint)
	}
}

func openInputFile() io.ReadCloser {
	if flagFilename == "-" {
		return os.Stdin
	}

	infile, err := os.Open(flagFilename)
	if err != nil {
		log.Fatalf("openInputFile: %v", err)
	}

	return infile
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

	infile := openInputFile()
	defer infile.Close()

	reader := csv.NewReader(infile)
	reader.FieldsPerRecord = 2

	cfg := config.Default()
	cfg.RPCURL = rpcAddr

	_, meta, err := setupConnection(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	firstLine, err := reader.Read()
	if err != nil {
		log.Fatal(err)
	}

	debug("senderAddress:", firstLine[0])

	_ = mustParseAddr(firstLine[0])
	calls := make([]types.Call, 0)

	var numErrors int // malformed lines

	for i := 1; ; i++ {
		r, err := reader.Read()
		if err != nil && err == io.EOF {
			debug("EOF")
			break
		} else if err != nil {
			log.Printf("error encountered while parsing line %d: %v", i, err)
			numErrors += 1
			continue
		}

		debug("recipient:", r[0], "amount:", r[1])

		addr, err1 := parseAddr(r[0])
		amt, err2 := parseAmount(r[1])

		if err1 != nil || err2 != nil {
			log.Printf("%d: parseAddr error: %v, parseAmount error: %v", i, err1, err2)
			numErrors += 1
			continue
		}

		c, err := types.NewCall(meta, "Balances.transfer", addr, types.NewUCompactFromUInt(amt))
		if err != nil {
			log.Printf("%d: error in creating a new call: %v", i, err)
			numErrors += 1
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

	// TODO: execute the extrinsic
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

func parseAddr(s string) (types.Address, error) { return types.NewAddressFromAccountID([]byte(s)) }

func parseAmount(s string) (uint64, error) { return strconv.ParseUint(s, 10, 64) }

func mustParseAddr(s string) types.Address {
	addr, err := parseAddr(s)
	if err != nil {
		log.Fatalf("parseAddr: %v", err)
	}

	return addr
}

func debug(v ...any) {
	debugLogger.Println(v...)
}

type dummyWriter struct{}

func (*dummyWriter) Write([]byte) (int, error) { return 0, nil }
