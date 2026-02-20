//go:build ignore

// Quick sandbox test for the Teller connector.
// Run: go run ./internal/connectors/plugins/public/teller/cmd/sandbox-test/
//
// This starts a local server that:
// 1. Serves a page with the Teller Connect widget (sandbox mode)
// 2. After enrollment, captures the access token
// 3. Tests the Teller API endpoints via our client
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	tellerclient "github.com/formancehq/payments/internal/connectors/plugins/public/teller/client"
)

const applicationID = "app_pov8nkrv5e98k7qnq8000"

func main() {
	port := "9876"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	c := tellerclient.New("teller-sandbox-test", true)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, indexHTML, applicationID)
	})

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.URL.Query().Get("access_token")
		enrollmentID := r.URL.Query().Get("enrollment_id")
		if accessToken == "" {
			http.Error(w, "missing access_token", 400)
			return
		}

		log.Printf("=== Enrollment complete ===")
		log.Printf("  access_token:  %s", accessToken)
		log.Printf("  enrollment_id: %s", enrollmentID)

		ctx := context.Background()

		// Test 1: List accounts
		log.Printf("\n--- List Accounts ---")
		accounts, err := c.ListAccounts(ctx, accessToken)
		if err != nil {
			log.Printf("ERROR listing accounts: %v", err)
			http.Error(w, fmt.Sprintf("list accounts failed: %v", err), 500)
			return
		}
		for _, a := range accounts {
			log.Printf("  Account: %s (%s/%s) at %s", a.ID, a.Type, a.Subtype, a.Institution.Name)
		}

		// Test 2: Get balances for each account
		log.Printf("\n--- Get Balances ---")
		for _, a := range accounts {
			bal, err := c.GetBalance(ctx, accessToken, a.ID)
			if err != nil {
				log.Printf("  ERROR balance for %s: %v", a.ID, err)
				continue
			}
			log.Printf("  Balance for %s: available=%s, ledger=%s", a.ID, bal.Available, bal.Ledger)
		}

		// Test 3: List transactions for each account
		log.Printf("\n--- List Transactions ---")
		for _, a := range accounts {
			txns, err := c.ListTransactions(ctx, accessToken, a.ID, "", 5)
			if err != nil {
				log.Printf("  ERROR transactions for %s: %v", a.ID, err)
				continue
			}
			log.Printf("  Transactions for %s (%d returned):", a.ID, len(txns))
			for _, tx := range txns {
				log.Printf("    %s | %s | %s | %s | %s", tx.ID, tx.Date, tx.Amount, tx.Description, tx.Status)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		result := map[string]interface{}{
			"status":        "ok",
			"accounts":      len(accounts),
			"access_token":  accessToken,
			"enrollment_id": enrollmentID,
		}
		json.NewEncoder(w).Encode(result)
	})

	log.Printf("Starting Teller sandbox test server on http://localhost:%s", port)
	log.Printf("Open the URL in your browser, click Connect, use credentials: username / password")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

const indexHTML = `<!DOCTYPE html>
<html>
<head><title>Teller Sandbox Test</title></head>
<body>
  <h1>Teller Sandbox Test</h1>
  <p>Click the button below to open Teller Connect (sandbox mode).</p>
  <p>Use credentials: <b>username</b> / <b>password</b></p>
  <button id="connect-btn" style="font-size:18px;padding:12px 24px;cursor:pointer">Connect Bank Account</button>
  <pre id="result" style="margin-top:20px;background:#f4f4f4;padding:16px;display:none"></pre>

  <script src="https://cdn.teller.io/connect/connect.js"></script>
  <script>
    var tellerConnect = TellerConnect.setup({
      applicationId: "%s",
      environment: "sandbox",
      onSuccess: function(enrollment) {
        console.log("Enrollment success:", enrollment);
        var params = new URLSearchParams({
          access_token: enrollment.accessToken,
          enrollment_id: enrollment.enrollment.id,
          institution: enrollment.enrollment.institution.name,
        });
        fetch("/callback?" + params.toString())
          .then(r => r.json())
          .then(data => {
            document.getElementById("result").style.display = "block";
            document.getElementById("result").textContent = JSON.stringify(data, null, 2);
          })
          .catch(err => {
            document.getElementById("result").style.display = "block";
            document.getElementById("result").textContent = "Error: " + err;
          });
      },
      onFailure: function(failure) {
        console.error("Enrollment failed:", failure);
        alert("Enrollment failed: " + JSON.stringify(failure));
      },
      onExit: function() {
        console.log("User exited Teller Connect");
      }
    });
    document.getElementById("connect-btn").addEventListener("click", function() {
      tellerConnect.open();
    });
  </script>
</body>
</html>
`
