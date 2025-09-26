package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	_ "embed"

	"github.com/coinbase-samples/prime-sdk-go/balances"
	"github.com/coinbase-samples/prime-sdk-go/client"
	"github.com/coinbase-samples/prime-sdk-go/credentials"
	"github.com/coinbase-samples/prime-sdk-go/portfolios"
	"github.com/coinbase-samples/prime-sdk-go/transactions"
	"github.com/coinbase-samples/prime-sdk-go/wallets"
)

func LoadCredentials() string {
	return os.Getenv("COINBASE_CREDENTIALS")
}

type Export struct {
	Accounts []Account `json:"accounts"`
}

type Account struct {
	Id          string            `json:"id"`
	AccountName string            `json:"accountName"`
	CreatedAt   string            `json:"createdAt"`
	Metadata    map[string]string `json:"metadata"`
	Balances    []Balance         `json:"balances"`
}

type Balance struct {
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
}

func main() {
	_credentials := LoadCredentials()
	if _credentials == "" {
		log.Fatalf("unable to load prime credentials")
	}

	primeCredentials, err := credentials.UnmarshalCredentials([]byte(_credentials))
	if err != nil {
		log.Fatalf("unable to load prime credentials: %v", err)
	}

	fmt.Println(primeCredentials)

	httpClient, err := client.DefaultHttpClient()
	httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	if err != nil {
		log.Fatalf("unable to load default http client: %v", err)
	}

	client := client.NewRestClient(primeCredentials, httpClient)

	portfoliosSvc := portfolios.NewPortfoliosService(client)
	balancesSvc := balances.NewBalancesService(client)
	transactionsSvc := transactions.NewTransactionsService(client)
	walletsSvc := wallets.NewWalletsService(client)

	export := Export{
		Accounts: []Account{},
	}

	res, err := portfoliosSvc.ListPortfolios(context.Background(), &portfolios.ListPortfoliosRequest{})

	if err != nil {
		log.Fatalf("unable to list portfolios: %v", err)
	}

	for _, p := range res.Portfolios {
		account := Account{
			Id:          p.Id,
			AccountName: p.Name,
			CreatedAt:   time.Now().Format(time.RFC3339),
			Metadata: map[string]string{
				"spec.coinbase.com/type":         "portfolio",
				"spec.coinbase.com/portfolio_id": p.Id,
			},
		}

		fmt.Printf("|-- %s: %s\n", p.Id, p.Name)

		{
			fmt.Println("\t|-- balances")
			r, _ := balancesSvc.ListPortfolioBalances(context.Background(), &balances.ListPortfolioBalancesRequest{
				PortfolioId: p.Id,
			})

			for _, b := range r.Balances {
				fmt.Printf("\t\t|-- %s: %s\n", b.Symbol, b.Amount)

				amount, _ := big.NewFloat(0).SetString(b.Amount)
				amount = amount.Mul(amount, big.NewFloat(100))

				account.Balances = append(account.Balances, Balance{
					Currency: fmt.Sprintf("%s/2", strings.ToUpper(b.Symbol)),
					Amount:   amount.String(),
				})
			}
		}

		{
			fmt.Println("\t|-- transactions")
			r, _ := transactionsSvc.ListPortfolioTransactions(context.Background(), &transactions.ListPortfolioTransactionsRequest{
				PortfolioId: p.Id,
			})

			for _, t := range r.Transactions {
				fmt.Printf(
					"\t\t|-- %s: %s [%s > %s] [wallet: %s] %s\n",
					t.Symbol,
					t.Amount,
					t.TransferFrom.Type,
					t.TransferTo.Type,
					t.WalletId,
					t.Type,
				)
			}
		}

		{
			fmt.Println("\t|-- wallets")
			r, _ := walletsSvc.ListWallets(context.Background(), &wallets.ListWalletsRequest{
				PortfolioId: p.Id,
			})

			for _, w := range r.Wallets {
				account := Account{
					Id:          w.Id,
					AccountName: w.Name,
					CreatedAt:   time.Now().Format(time.RFC3339),
					Metadata: map[string]string{
						"spec.coinbase.com/type":        "wallet",
						"spec.coinbase.com/wallet_type": w.Type,
					},
				}
				export.Accounts = append(export.Accounts, account)

				fmt.Printf("\t\t|-- %s: %s (%s)\n", w.Id, w.Name, w.Type)

				// {
				// 	r, err := balancesSvc.GetWalletBalance(
				// 		context.Background(),
				// 		&balances.GetWalletBalanceRequest{
				// 			Id: w.Id,
				// 		},
				// 	)

				// 	if err != nil {
				// 		log.Printf("unable to get wallet balance: %v", err)
				// 		continue
				// 	}

				// 	fmt.Println(r.Balance.Amount, r.Balance.Symbol)
				// }
			}
		}

		export.Accounts = append(export.Accounts, account)
	}

	file, err := os.Create("export.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	json.NewEncoder(file).Encode(export)
}
