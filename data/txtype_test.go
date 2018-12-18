package data

import (
	"testing"
)

func TestTxEntries(t *testing.T) {
	TxEntries := []TxEntry{{"alice", "add"}, {"bob", "add"}, {"mary", "remove"}}
	for _, j := range TxEntries {
		if LookupTxEntry(&j) {
			t.Error("LookupTxEntry returned true, wanted false")
		} else {
			err := InsertTxEntry(&j)
			if err != nil {
				panic(err)
			}
		}
	}
	for _, j := range TxEntries {
		if !LookupTxEntry(&j) {
			t.Error("LookupTxEntry returned false, wanted true")
		}
	}
	for _, j := range TxEntries {
		DeleteTxEntry(&j)
	}
	for _, j := range TxEntries {
		if LookupTxEntry(&j) {
			t.Error("LookupTxEntry returned true, wanted false")
		}
	}
}
func TestGetTxEntries(t *testing.T) {
	TxEntries := []TxEntry{
		{"yonglupo", "add"},
		{"floretta", "add"},
		{"hassieve", "add"},
		{"yokoshum", "add"},
		{"stacysle", "add"},
		{"kandacel", "add"},
		{"sidneyra", "add"},
		{"pauletta", "add"},
		{"mozellah", "add"},
		{"shirleew", "remove"},
		{"larondam", "add"},
		{"joshsmul", "add"},
		{"margenec", "remove"},
		{"lerateff", "add"},
		{"latoyiah", "add"},
		{"weldonva", "add"},
		{"carlynfr", "remove"},
		{"fannieal", "add"},
		{"charlynh", "add"},
		{"ladawncl", "add"},
		{"velvetla", "add"},
		{"kelleyti", "remove"},
		{"clifford", "add"},
		{"catricer", "remove"},
		{"jimmiege", "remove"},
		{"justaloc", "add"},
		{"micahbin", "add"},
		{"johnatha", "add"},
	}
	for _, j := range TxEntries {
		if LookupTxEntry(&j) {
			t.Error("LookupTxEntry returned true, wanted false")
		} else {
			err := InsertTxEntry(&j)
			if err != nil {
				panic(err)
			}
		}
	}
	for i := 0; i < 4; i++ {
		entries, err := GetTxEntries()
		if err != nil {
			panic(err)
		}
		switch i {
		case 0:
			want := 10
			if got := len(entries); got != want {
				t.Errorf("GetTxEntries returned %d entries, wanted %d\n", got, want)
			}
		case 1:
			want := 10
			if got := len(entries); got != want {
				t.Errorf("GetTxEntries returned %d entries, wanted %d\n", got, want)
			}
		case 2:
			want := 8
			if got := len(entries); got != want {
				t.Errorf("GetTxEntries returned %d entries, wanted %d\n", got, want)
			}
		case 3:
			want := 0
			if got := len(entries); got != want {
				t.Errorf("GetTxEntries returned %d entries, wanted %d\n", got, want)
			}
		}
		for _, j := range entries {
			DeleteTxEntry(&j)
		}

	}

}
