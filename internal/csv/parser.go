package csv

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

const (
	NumFields = 2

// SenderFollowedByBlankLine = true
)

type Record interface {
	Address() types.Address
	Amount() uint64
	ZeroAmount() bool
}

type rawRecord struct {
	addr types.Address
	amt  uint64
}

func newRawRecord(s, a string) (*rawRecord, error) {
	addrT, err := parseAddr(s)
	if err != nil {
		return nil, err
	}

	amt, err := strconv.ParseUint(a, 10, 64)
	if err != nil {
		return nil, err
	}

	return &rawRecord{addr: addrT, amt: amt}, nil
}

func (r *rawRecord) Address() types.Address { return r.addr }
func (r *rawRecord) Amount() uint64         { return r.amt }
func (r *rawRecord) ZeroAmount() bool       { return r.amt == 0 }

type feed struct {
	csvReader  *csv.Reader
	chewed     int
	numFields  int
	eof        bool
	err        error
	senderAddr types.Address
}

func newFeed(r io.Reader, fields int) *feed {
	return &feed{csvReader: csv.NewReader(r), numFields: fields}
}

// Default returns a batch parser initialized with configuration's values.
func Default() Reader { return New(emptyBuffer(), NumFields) }

// SetInpput sets the the parser's input feed to a io.Reader instance.
func (f *feed) SetInput(r io.Reader) { f.csvReader = csv.NewReader(r) }

// New returns a batch parser that reads records from a io.Reader.
func New(r io.Reader, n int) Reader { return newFeed(r, n) }

// SetNumFields
func (f *feed) SetNumFields(n int) {
	f.numFields = n
}

func (f *feed) Reset() {
	f.csvReader = csv.NewReader(emptyBuffer())
	f.chewed = 0
	f.numFields = NumFields
	f.eof = false
}

func (f *feed) EOF() bool { return f.eof }

func (f *feed) Sender() (string, error) {
	if f.EOF() {
		return "", f.err
	}

	if f.chewed > 0 {
		return "", fmt.Errorf("sender line was already read")
	}

	record, err := f.csvReader.Read()
	if err != nil {
		f.eof = err == io.EOF
		f.err = err
		return "", err
	}

	f.chewed += 1

	return record[0], nil
}

type Reader interface {
	Reset()
	Read() (Record, error)
	SetInput(r io.Reader)
	SetNumFields(n int)
	EOF() bool
	Sender() (string, error)
}

func NewReader(r io.Reader) Reader {
	csvReader := csv.NewReader(r)
	return &feed{csvReader: csvReader, numFields: 2}
}

// Record reads the next record.
func (f *feed) Read() (Record, error) {
	if f.chewed == 0 {
		return nil, fmt.Errorf("read the sender address first")
	}

	if f.EOF() {
		return nil, io.EOF
	}

	line, err := f.csvReader.Read()
	if err != nil {
		f.err = err

		if err != io.EOF {
			f.chewed += 1
		}

		return nil, f.err
	}

	return newRawRecord(line[0], line[1])
}

// Records returns the number of the records read.
func (f *feed) Records() int { return f.chewed }

func emptyBuffer() *bytes.Buffer { return bytes.NewBuffer(make([]byte, 0)) }

func parseSenderRecord(record []string) (types.Address, error) {
	var v string

	for _, v = range record {
		if v != "" {
			return parseAddr(v)
		}
	}

	return types.Address{}, ErrEmptySenderLine
}

func parseAddr(s string) (types.Address, error) { return types.NewAddressFromAccountID([]byte(s)) }

var (
	ErrEmptySenderLine = errors.New("csv: couldn't find a valid sender address as the line is empty")
)
