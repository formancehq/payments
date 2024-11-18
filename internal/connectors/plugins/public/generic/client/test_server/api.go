package testserver

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/genericclient"
)

type API struct {
	firstTimeCreation time.Time
	nbAccounts        int
}

func (a *API) accountsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, pageSize, createdAtFrom, err := getQueryParams(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		switch a.firstTimeCreation.Compare(createdAtFrom) {
		case -1, 0:
			api.Ok(w, []genericclient.Account{})
			return
		default:
		}

		index := getIndex(a.firstTimeCreation, createdAtFrom, a.nbAccounts, page, pageSize)
		if index == nil {
			api.RawOk(w, []genericclient.Account{})
			return
		}

		accounts := make([]genericclient.Account, 0)
		for i := *index; i < a.nbAccounts && len(accounts) < pageSize; i++ {
			accounts = append(accounts, genericclient.Account{
				Id:          strconv.Itoa(i),
				AccountName: fmt.Sprintf("Account %d", i),
				CreatedAt:   a.firstTimeCreation.Add(time.Duration(-a.nbAccounts+i) * time.Minute),
				Metadata: map[string]string{
					"foo": "bar",
				},
			})
		}

		api.RawOk(w, accounts)
	}
}

func (a *API) balancesList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api.RawOk(w, genericclient.Balances{})
	}
}

func (a *API) beneficiariesList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api.RawOk(w, []genericclient.Beneficiary{})
	}
}

func (a *API) transactionsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api.RawOk(w, []genericclient.Transaction{})
	}
}

func getIndex(firstTimeCreation, createdAtFrom time.Time, nbObjects, page, pageSize int) *int {
	d := math.Floor(firstTimeCreation.Sub(createdAtFrom).Minutes())
	index := nbObjects - int(d)
	if index < 0 {
		index = 0
	}
	index = index + (page-1)*pageSize
	if index > nbObjects {
		return nil
	}

	return pointer.For(index)
}

func getQueryParams(r *http.Request) (int, int, time.Time, error) {
	page := r.URL.Query().Get("page")
	pageSize := r.URL.Query().Get("pageSize")
	createdAtFrom := r.URL.Query().Get("createdAtFrom")

	resPage, err := strconv.Atoi(page)
	if err != nil {
		return 0, 0, time.Time{}, err
	}

	resPageSize, err := strconv.Atoi(pageSize)
	if err != nil {
		return 0, 0, time.Time{}, err
	}

	resCreatedAtFrom := time.Time{}
	if createdAtFrom != "" {
		resCreatedAtFrom, err = time.Parse(time.RFC3339Nano, createdAtFrom)
		if err != nil {
			return 0, 0, time.Time{}, err
		}
	}

	return resPage, resPageSize, resCreatedAtFrom, nil
}
