package csv

// import (
// 	"bytes"
// 	"encoding/csv"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"strconv"

// 	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
// )

// const (
// 	NumFields = 2

// // SenderFollowedByBlankLine = true
// )

// type Record interface {
// 	Address() types.Address
// 	Amount() uint64
// 	ZeroAmount() bool
// 	Values() []string
// }

// type feed struct {
// 	csvReader     *csv.Reader
// 	chewed        int
// 	numFields     int
// 	eof           bool
// 	err           error
// 	senderAddr    types.Address
// 	senderRawAddr string
// }

// func newFeed(r io.Reader, fields int) *feed {
// 	return &feed{csvReader: csv.NewReader(r), numFields: fields}
// }

// // New returns a batch parser that reads records from a io.Reader.
// func NewScanner(r io.Reader) Scanner { return newFeed(r, NumFields) }

// // SetInpput sets the the parser's input feed to a io.Reader instance.
// func (f *feed) Init() {
// 	f.csvReader = csv.NewReader(r)
// }

// func (f *feed)

// // Sender returns the sender address. It retrns an error if called
// func (f *feed) Sender() types.Address() {
// 	if f.senderRawAddr != "" {
// 		return f.senderAddr
// 	}

// 	record, err := f.csvReader.Read()
// 	if err != nil {
// 		f.err = err
// 		return "", err
// 	}

// 	f.chewed += 1

// 	f.senderRawAddr = record[0]

// 	return record[0], nil
// }

// type Reader interface {
// 	Reset()
// 	Read() (Record, error)
// 	SetInput(r io.Reader)
// 	SetNumFields(n int)
// 	EOF() bool
// 	Sender() (string, error)
// }

// type Scanner interface {
// 	Sender() types.Address
// 	Record() Record
// 	Err() error
// }

// func NewReader(r io.Reader) Scanner {
// 	csvReader := csv.NewReader(r)
// 	return &feed{csvReader: csvReader, numFields: 2}
// }

// // Read reads and parses the next record.
// func (f *feed) Scan() bool {
// 	rec, err := f.readRaw()
// 	if err != nil {
// 		return nil, fmt.Errorf("Read: %v", err)
// 	}

// 	err = rec.parse()

// 	if err != nil {
// 		err = fmt.Errorf("Read: %v", err)
// 	}

// 	return rec, err
// }

// // Read reads and parses the next record.
// func (f *feed) Read() (Record, error) {
// 	rec, err := f.readRaw()
// 	if err != nil {
// 		return nil, fmt.Errorf("Read: %v", err)
// 	}

// 	err = rec.parse()

// 	if err != nil {
// 		err = fmt.Errorf("Read: %v", err)
// 	}

// 	return rec, err
// }

// func (f *feed) readRaw() (*rawRecord, error) {
// 	if f.chewed == 0 {
// 		return nil, fmt.Errorf("read the sender address first")
// 	}

// 	if f.EOF() {
// 		return nil, io.EOF
// 	}

// 	line, err := f.csvReader.Read()
// 	if err != nil {
// 		f.err = err

// 		if err != io.EOF {
// 			f.chewed += 1
// 		}

// 		return nil, f.err
// 	}

// 	return newRecord(line), nil
// }

// // Records returns the number of the records read.
// func (f *feed) Records() int { return f.chewed }

// func emptyBuffer() *bytes.Buffer { return bytes.NewBuffer(make([]byte, 0)) }

// func parseSenderRecord(r []string) (types.Address, error) {
// 	var v string

// 	for _, v = range r {
// 		if v != "" {
// 			return parseAddr(v)
// 		}
// 	}

// 	return types.Address{}, ErrEmptySenderLine
// }

// func parseAddr(s string) (types.Address, error) { return types.NewAddressFromAccountID([]byte(s)) }

// var (
// 	ErrEmptySenderLine = errors.New("csv: couldn't find a valid sender address as the line is empty")
// )

// type rawRecord struct {
// 	vals []string
// 	addr types.Address
// 	amt  uint64
// }

// func newRecord(vals []string) *rawRecord {
// 	return &rawRecord{vals: vals}
// }

// func (r *rawRecord) parse() error {
// 	if len(r.vals) < 2 {
// 		return errors.New("insufficient number of fields")
// 	}

// 	addrT, err := parseAddr(r.vals[0])
// 	if err != nil {
// 		return err
// 	}

// 	r.addr = addrT

// 	amt, err := strconv.ParseUint(r.vals[1], 10, 64)
// 	if err != nil {
// 		return err
// 	}

// 	r.amt = amt

// 	return nil
// }

// func (r *rawRecord) Address() types.Address { return r.addr }
// func (r *rawRecord) Amount() uint64         { return r.amt }
// func (r *rawRecord) ZeroAmount() bool       { return r.amt == 0 }
// func (r *rawRecord) Values() []string       { return r.vals }
