package csv

import (
	"bytes"
	"encoding/csv"
	"os"
	"reflect"
	"testing"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

func Test_rawRecord_parse(t *testing.T) {
	tests := []struct {
		name    string
		fields  []string
		wantErr bool
	}{
		{"nil fields slice", nil, true},
		{"one field", []string{"5F24AhXc4tckrVrsA3E5n7jpXQTX6PZ572xMe7wDG8gtEPhu"}, true},
		{"two fields w malformed one", []string{"5F24AhXc4tckrVrsA3E5n7jpXQTX6PZ572xMe7wDG8gtEPhu", "invalid"}, true},
		{"happy path", []string{"5F24AhXc4tckrVrsA3E5n7jpXQTX6PZ572xMe7wDG8gtEPhu", "120"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRecord(tt.fields)
			if err := r.parse(); (err != nil) != tt.wantErr {
				t.Errorf("rawRecord.parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(r.Values()) != len(tt.fields) {
				t.Errorf("rawRecord.Values() = %v, wanted %v", r.Values(), tt.fields)
			}
		})
	}
}

func Test_rawRecord_Address(t *testing.T) {
	type fields struct {
		vals []string
		addr types.Address
		amt  uint64
	}
	tests := []struct {
		name   string
		fields fields
		want   types.Address
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rawRecord{
				vals: tt.fields.vals,
				addr: tt.fields.addr,
				amt:  tt.fields.amt,
			}
			if got := r.Address(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rawRecord.Address() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser(t *testing.T) {
	r := New(bytes.NewBufferString(readFile(t, "malformed_sender.csv")), NumFields)
}

func readFile(t *testing.T, filename string) string {
	t.Helper()

	contents, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	return string(contents)
}

func Test_feed_Sender(t *testing.T) {
	type fields struct {
		csvReader  *csv.Reader
		chewed     int
		numFields  int
		eof        bool
		err        error
		senderAddr types.Address
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &feed{
				csvReader:  tt.fields.csvReader,
				chewed:     tt.fields.chewed,
				numFields:  tt.fields.numFields,
				eof:        tt.fields.eof,
				err:        tt.fields.err,
				senderAddr: tt.fields.senderAddr,
			}
			got, err := f.Sender()
			if (err != nil) != tt.wantErr {
				t.Errorf("feed.Sender() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("feed.Sender() = %v, want %v", got, tt.want)
			}
		})
	}
}
