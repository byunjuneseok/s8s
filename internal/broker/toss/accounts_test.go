package toss

import (
	"context"
	"net/http"
	"testing"

	"github.com/byunjuneseok/s8s/internal/domain"
)

func TestAccounts(t *testing.T) {
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != accountsPath {
			t.Errorf("path = %q, want %q", r.URL.Path, accountsPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":[
			{"accountNo":"123-45-678","accountSeq":7,"accountType":"BROKERAGE"},
			{"accountNo":"999-00-111","accountSeq":8,"accountType":"BROKERAGE"}
		]}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Accounts(context.Background())
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d accounts, want 2", len(got))
	}
	want := domain.Account{No: "123-45-678", Seq: 7, Type: domain.Brokerage}
	if got[0] != want {
		t.Errorf("account[0] = %+v, want %+v", got[0], want)
	}
	if got[1].Seq != 8 {
		t.Errorf("account[1].Seq = %d, want 8", got[1].Seq)
	}
}
