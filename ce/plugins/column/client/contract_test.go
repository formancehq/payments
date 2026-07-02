//go:build contract

// Package client contract test for the Column connector.
//
// This is a CONTRACT test: it calls the real Column sandbox over the network
// through the same client.Client the connector uses, and asserts that the
// responses the Payments project depends on have not drifted in schema (field
// presence + types) or in list ordering. It is gated behind the `contract`
// build tag so it never runs as part of `just tests` (which only enables
// `-tags it`); it runs daily via the contract-tests GitHub workflow.
//
// Run locally:
//
//	COLUMN_CONTRACT_API_KEY=test_... just contract-tests column
//
// The sandbox and production share the api.column.com host, so the endpoint is
// hardcoded (contractEndpoint) — a test_-prefixed key selects the sandbox.
// Without COLUMN_CONTRACT_API_KEY the suite Skips rather than fails, so it is
// safe to run anywhere.
//
// Money movement: the InitiateTransfer / InitiatePayout / CreateCounterParty
// specs make REAL calls against the sandbox at the smallest possible amount
// (1 minor unit) with AllowOverdraft, deriving the account/counterparty IDs
// from the list reads and using a unique idempotency Reference per run. They
// accumulate sandbox state by design (accepted). ReversePayout is intentionally
// NOT exercised: it sets no idempotency key and needs a settled ACH transfer,
// so it cannot be safely repeated daily — its schema is covered by unit tests.
package client

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/formancehq/payments/pkg/domain/contracttest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestColumnContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Column Contract Suite")
}

const contractPageSize = 100

// contractEndpoint is the Column API host. Sandbox and production share this
// host; a test_-prefixed COLUMN_CONTRACT_API_KEY selects the sandbox.
const contractEndpoint = "https://api.column.com"

// contractEventSubscriptionURL is a syntactically valid public URL used only to
// create the temporary webhook endpoint. NOTE: real end-to-end webhook
// *delivery* (Column POSTing to this URL and us verifying its signature) is NOT
// exercised here — CI has no public ingress. This contract test only validates
// the management-API representation of the webhook endpoint (create, shape,
// delete). Delivery/signature verification is covered by the unit tests.
const contractEventSubscriptionURL = "https://example.com/api/payments/v3/connectors/webhooks/column-contract"

// expectedAccountIDs / expectedCounterpartyIDs pin the known, seeded order of
// the sandbox's accounts and counterparties. They start empty; the schema specs
// print the live IDs to stderr as a paste-ready Go literal (bootstrap log) so a
// maintainer can fill these in to enable the ordering contract.
var (
	expectedAccountIDs = []string{
		"bacc_2stc6SrkwuFtqae0JnkFq5Ilp87",
		"bacc_2stc3DTRnzMnk3JlG9EaP8WT9i1",
		"bacc_2stc3BwH8MsAz6ApyC4Zet8rmEO",
		"bacc_2stc39sSeBOgZTkpHXOS5Wbf6yq",
	}
	expectedCounterpartyIDs = []string{
		"cpty_3FtW4VAzzojgw92rBUW2pY5Be4u",
		"cpty_3FtV4t4aPv8WpLlS8ERLSYv17C4",
		"cpty_3FrgmmAbJ2238zJV5sag6B1igBU",
		"cpty_3Fred5pQo9WLMVjhtckcnQPMhzv",
		"cpty_3FreUB5fnQox9OfrqyawYCm1qy4",
		"cpty_2xMLcqh1EhizOc4NJQDtxAWi0Uz",
		"cpty_2wrQCYTyQQCuKVvjXkcVOS07H3x",
		"cpty_2wjDdKBW1d9qUw8DH31dxK4qLmC",
		"cpty_2wjC9x13extBSIY6U9gMwMZQZXh",
		"cpty_2wj2Ts0iCIRIt137CQyrhA9l6WA",
		"cpty_2wipvmAHxIDs3tUC38ZswkJntIo",
		"cpty_2wRn2nEgFauiZeAd9C612ZGQjq2",
		"cpty_2wRSnXhf2qDwqD9VP12VHz3lL6v",
		"cpty_2wPTp6wc2xbE1kbIRbPgE9JLqT7",
		"cpty_2wPQOGZ256tRvLGviRLJfXOhX9g",
		"cpty_2wPMH2UohOtSTHuvcUzDBd4rIIJ",
		"cpty_2uxMOgeZ3wmdsZItEzj6dkUn9JP",
		"cpty_2ue7xakrxHavyeWoDg6YDhz4LMS",
		"cpty_2udhKZvi89PsbrPLwsNPVsutQLF",
		"cpty_2udcZKBUpb7FAovn1jHHm4uyPTG",
		"cpty_2udYxYAtqCwueKw0RRf8kxNu7zc",
		"cpty_2udUj70Jspvpyu4VHK4l6lgvaKB",
		"cpty_2ud0LSvxluhM39ATU5LuvAj0j96",
		"cpty_2ucmC1EMijpREgYLsluRiZZTJ90",
		"cpty_2ucm3IoBNmnq9qvNWjMVWtZ1UMr",
		"cpty_2ucd6lvBUiGYmCNDBf9bVStycSb",
		"cpty_2ucYx9jlCjGiA6DcZuPdjsj213a",
		"cpty_2ucX7qiTb13MTLegRJIs63dxv0n",
		"cpty_2uXXUzbPuwmHPpDG2TrOkdS3h8y",
		"cpty_2uXXMp5pTeCumcb77wKaykyYvw3",
		"cpty_2uXUxA8EOgNTtqw9d08QCaZQYFP",
		"cpty_2uXUpMzxVr8MtjAsgy6Y9AKm6St",
		"cpty_2uIv9BaU7sxyQcFmqdLqqcDMhN7",
		"cpty_2uH1IwpOJzkohXECOFDH1yGAkZt",
		"cpty_2uGwCYsEvcTkeSsNAEF7TOIEQRn",
		"cpty_2uGvSEIqHYLVhqJgDfiTNp9FwCR",
		"cpty_2uEOM88ti2AbVKekWPQDETBimUZ",
		"cpty_2uEGKqsVlo5xnEhvJqk2L0XvrR4",
		"cpty_2tzhVp5uQzKYpiAjZ3tVbJlKeQ1",
		"cpty_2tzeWd0LKiggJnvViUVRu6u5vKl",
		"cpty_2tzJnLhf5yDFMoFufXGF8ZeIjzb",
		"cpty_2tzGK7h3dmW0qa14Iy72jJ4joPE",
		"cpty_2tzEcQPgWpp4MegfURAfj0TE1pO",
		"cpty_2tz8eBuwB1pGW3YK0Te2hPE7Elo",
		"cpty_2txljn6Kv5Xir8bUll9txoXtLbc",
		"cpty_2txYUjoMa7q3yHDKn6R43XwT6Pc",
		"cpty_2ttukkb4phX0LqfXTKzKi4aDESy",
		"cpty_2tnqtj2S6CrbA8sUBEEhBWDOW0f",
		"cpty_2tnS75AeBpmSgV3Gbnm97ydSW35",
		"cpty_2tgBsGxbrfA1wCxS02gee7AWUrR",
		"cpty_2tUwokFcHgDcuCZ9gnCXiWcU7fl",
		"cpty_2stcDuu0FNQngAkcogWPZbKTChq",
		"cpty_2stc9ESzLhT2rrlrFK0BpWrak14",
	}
	// expectedTransactionIDs pins the known transactions in the chronological
	// order the connector consumes them (the Timeline walk). Starts empty; set
	// COLUMN_CONTRACT_BOOTSTRAP=1 and fill from the bootstrap log to enable.
	expectedTransactionIDs = []string{
		"wire_2stc9DzRzCPIz3lunRCgTo9MF6D",
		"swft_2stcE18myt6dl7m5SspfMSDpu8G",
		"acht_2stcTPFekE2apsqseMyPCcp20qB",
		"book_2steshUjZjTDntahMEQF9wyYsYf",
		"wire_2tzNhSpuU1b29GrTYuVIyr1XZp1",
		"wire_2tzhcYe7X65k1ZQSV2yyCVya4tg",
		"swft_2tzntg9QMtlCVJXBCrcBrJy3Kn2",
		"acht_2tzpLxtAQhxYcytAmVML7Cinn1I",
		"swft_2tzqTacjtbItJWw6bFKbStKhLDe",
		"swft_2tzqtDrDQ5y9OBc0BboUehBmntG",
		"rttr_2tzs0AkyDouVNOWYB7nsYNt2I6d",
		"rttr_2tzuRxGx2h6TWTGZGWCzHGlzgvO",
		"rttr_2tzuXrBDSNhgUgmnSZsHhB6vvvq",
		"wire_2tzuywSecqv1ZZ5bGAEnlochKmZ",
		"acht_2u08kxuI5Y3iZo06gAl0sPMsr2I",
		"acht_2u09okQz7XQOfqtURvgayqOrEJz",
		"book_2u0DNv73xfffkriHZE3NDnts80b",
		"book_2u0ENmSJYXfybjcgkSfYbpL5cOb",
		"acht_2u0Es5wn1wmNf2ZL4D1K0XunzMi",
		"book_2u0EyIJpGHyZcoE5d0nfAkKR72i",
		"acht_2u0FYuuyMLfhynQD7ugd7dHOHhc",
		"acht_2u0FZsHYXfFzJcrmajpdkDtRsF6",
		"book_2u0H6rBiZQikSaZONehosxjYPdO",
		"acht_2u0NyJbNrEkAFDYPhgWchRfXwHW",
		"acht_2u0RJe28mJc2EtC19EDnbBEspmy",
		"wire_2u2jRdRiJDyEv7MH54qb6XrEkzd",
		"swft_2u2lOGu1uNmxq8MMKENUHYdBXXa",
		"acht_2u2lml8wGKmXfZ95ucMvUVGaGQx",
		"rttr_2u2loDmRpTyZP0Rt8z12NXMa0u3",
		"book_2u2mLTyQe1oNK3wV2TxTZxN7Gei",
		"book_2u2mrJycCVM7whxGXKXWkWINKaK",
		"acht_2uBPyAIu5L5H68RFId3AWYIr9TF",
		"book_2uETfqTF1mFkoQ34oGg2OLUiu2d",
		"book_2uH35ZT5dcORPgvUhE2Mrku7cfM",
		"acht_2uIptJlAa1w8F12yypOc5Fb8bSX",
		"acht_2uIuOWGGyxnuL19wKEm7R9VSAqL",
		"acht_2uIw4dF6j9ekAQmqEeq0B28gXaM",
		"acht_2uIyJCQ9DRf4r2WxjjRWjN7UaNr",
		"acht_2uJ0s1UjAYy1tR4yZxcohwv8IIL",
		"book_2uJ3Dwjd4R23oWvKDSbgflvpB3a",
		"acht_2uJ6OXBmbEiU4VSlOFC0Bih5U5y",
		"acht_2uJ8TGuXMXZrtg6BySUWKz3462M",
		"acht_2uJ9Mwl8M2qh2UQ8SEo0cRBbHyq",
		"acht_2uJB6vinQfiywnlDsUZ0VLgoxdo",
		"wire_2uJFnJZW648QxYUyB6BU7Fz269x",
		"wire_2uJGqnsiFVd5ZMRuywAnGe3S4mD",
		"rttr_2uJOC8zAvEslexKQy6WzFODhl5E",
		"rttr_2uJPFG6piu04OlHiWVSYavXYAdk",
		"swft_2uJSmaToYPEmepyWiy5WsKE5zfD",
		"swft_2uJT8h7AF7viQzRsZWG68si98tc",
		"swft_2uJTb9Q4WIgU2Z3ChdUQrmJN8cC",
		"swft_2uJVVY4ZvFbAmSU8rfVt1LtRlBM",
		"acht_2uJVcoh0gE0DrrpmboTNAYYMbJU",
		"acht_2uJm3GUqGhzcfb1F3O6VDogVRpb",
		"acht_2uJp60aocDjcm1FWJQEigmh37dm",
		"acht_2uJpLuBZoYqGj9GphwKwow9Y2mW",
		"acht_2uJpS9GN2zUbjvtfQD3XHPDZm7p",
		"acht_2uJsf5UQU9oHk0fkE0eQEmx4gfq",
		"acht_2uJsonyi8ypnzoQcNH2s2hQJWPQ",
		"acht_2uJtg1HRiX1s5zBFDztUxmS2Ii9",
		"acht_2uJu2m8w3Ujxw4coPkij9l5sKBv",
		"acht_2uJuOuE3URlkeL0faiKjZxDPJrC",
		"acht_2uJw69nsi3LAbnOMiJkEsQ2VGRr",
		"wire_2uJxD06yPWMpE6PJvw5zfbnyNTF",
		"swft_2uJy8LGH6eLXVX3lMtiP1serxkx",
		"rttr_2uJyHQjWFJNLHQmZbSe0LujReId",
		"swft_2uRosTJ3Irntac2uj9L92D0tGf4",
		"swft_2uSDgrSNjScr30Ek4i4iEFWHbVt",
		"book_2uXYpv9xrqOKUJttKpTNGtpM8Wo",
		"book_2ucZlp8anVWXPLBRMly3mtTSGoR",
		"acht_2ucc3pB66uQeIUKfX3Xn6XD4dai",
		"acht_2ucejf7l5mV2dJENZUnyQWYITQn",
		"acht_2ucfv0Y2297ZsBhN1bTmpZMlpT6",
		"wire_2uchAyaOA3Mj8riqiHhZbTfp2CM",
		"acht_2uckkYVHw7xD4Fdrhtn1yCCQo65",
		"swft_2uckv7GRuV9r68MCtnu09iHjHLG",
		"acht_2ucmoLy45mX5luFzlHequLUd9wK",
		"swft_2ucnb8bgQhSWVZqSR0IAO5mSS0y",
		"acht_2ucoVPkrD7tzVvLNhBpp7uyz0mA",
		"book_2ucq6BqD5SLYTYyLJZCUWx5LYCV",
		"book_2ucqjAOhTdF2gUP5mX4kC9vUSs5",
		"wire_2ucsEQKy6zc5jeACB9WJqoxpMQ9",
		"rttr_2ucsYNznL6gscWlCGJp3a3WyGNX",
		"rttr_2ucsmjNysvUf0kKw4GqrRFQAS3P",
		"rttr_2ud1a3fDAS91IATTOrj3FqsQE8t",
		"rttr_2ud51Gi432CAAo7PG1xFW2LNSzI",
		"rttr_2ud6mrhlN2C2mxeOthmxmvwNEQg",
		"wire_2ud7LRaA1IkdVriQHkYBAaFAEae",
		"acht_2ud7fXk3ToKGAq9UKjn0NcAfAsj",
		"rttr_2udD5bcpLpOEgsOdiGHQJMqvuCD",
		"wire_2udDMq1bEMOEO9WOJSX7UWHcGHX",
		"acht_2udDh8WLc1Mp4J9UMnmks8Utqt6",
		"swft_2udE8c3VpU8vmT7d1TloVu75sgw",
		"book_2udFAn8RR2nbxHJEQIYMI7VUtat",
		"book_2udGhZCnjaFa4mzdAMv19LG6cV6",
		"book_2udHiOopqasnWAl0EAIuMTGubd6",
		"swft_2udI7o98atuzEKDkP1lNB3aaLVD",
		"swft_2udIuhfKuVWDojFOUAVbizsPUq6",
		"acht_2udKKeOQl6L13gerKRvpVEnVGnl",
		"acht_2udN8cyTa7hCpLa9g5t1XGdmy5X",
		"book_2udRZfUINKqB1TaZyPxn9gCPTSL",
		"acht_2udS7NuPFcvsjfpYsj2uBZQWY8I",
		"acht_2udT2Z3IGyNlOOqv9SM4sccHQwO",
		"acht_2udT6siGVzhxfkenf9meu70JrJs",
		"acht_2udTAluX2nDULHGnbGb65ELf9Kw",
		"acht_2udUDChQBUIBnpUI1Ps1zMx0Shj",
		"book_2udZjz5DWkDpTWjx2TUJ2F4Rd0S",
		"book_2udc0Hkql03N6dsLwfEHlcYVAcU",
		"acht_2uddafLHeBqSIxEXLzfJ93SMWFO",
		"book_2udhsjhPS5WNXmhhTN55VaI0yz2",
		"acht_2udii5ClkryAggzvZihIpaTU5Eq",
		"acht_2udmgnDbL99dro2GiOK8wgxLRwP",
		"acht_2udnYiiL0CJ0qdfJcfSR8Hyt6mL",
		"acht_2udoFFpUFsqtOfoU3tecnf8hTQa",
		"wire_2udoQpziB4f3sH3e7REaepBMQBo",
		"wire_2udoon3ZAoIwJqX3JUaiZJ1cU1S",
		"rttr_2udpCLHGvwyIY7nfDWs2V6dFCyH",
		"rttr_2udpmPhlAARXmyWNEnu2XWkwPSo",
		"swft_2udqZG46OfFN5V1GfGqEo0OWy1P",
		"acht_2udrF4fdgqUeFDMqADKEEgC3ZyJ",
		"wire_2udrPU91T4zNCy0tVrK94u5v4HZ",
		"book_2udxFHv10f7jRvgjAH4u5HIFTTO",
		"acht_2udyBjxybeznUDUt8fdoD3Ulxd0",
		"wire_2ue0R7OSC0IumCQ7lg0FHCNMWgE",
		"rttr_2ue0kjBxW78gXkf6mCC7k0J0En3",
		"swft_2ue18ZPQ0ZXLUS90jmG7SAPWF2U",
		"acht_2ue1Ong1XXolEJrZUnNXUbKbjNF",
		"swft_2ue1fpszOktZFpcJBr4e5YwlpCe",
		"book_2ue2MZvtXeoOpO2ChqxnMVbxYqc",
		"swft_2ue3FsIT7ikFGaYAwwT7eEMK29U",
		"swft_2ue4BX6HzXhc5V8BMswb1B8BFQW",
		"rttr_2ue5TKJLH8D39WPvxQXToY1G5JL",
		"wire_2ue5uKx2p9nDErMfOkEw8GuVNoT",
		"acht_2ue66CgaFPpXb1G9G8AgTUPPFWh",
		"acht_2ue72GJIacImmyTQ25aunCKyDqQ",
		"book_2ue7nfVbrQLX6LhiA1zLeJl5hX8",
		"swft_2urO23ny6Lu6tkEo2ffpxRpGsaY",
		"book_2uwkeH8z6c4namEn38lC0p5UlTu",
		"acht_2uwvJHDEEAkKdLaMoxIR5GPdofg",
		"acht_2uwxCvNgwPy6ErDvc1hs5Le32Sb",
		"acht_2uwxgCx81KthZLhOyzEQkVjm115",
		"acht_2uwxmOMZGnssVYatWPPckb1Ur5R",
		"acht_2uwyOEAQ35G9bGkfDlwhYq5vr2S",
		"acht_2uwz3meH2kJ633FxgiiQrAdZpSV",
		"acht_2ux0FkKBJH6wIjoOykopGWsOfep",
		"acht_2ux0KJ7e0QPXBPB4sXlPDCJ5Hd5",
		"acht_2ux27Qe3dbH6tRPLal8ByFLuUPu",
		"acht_2ux8kKfYKV0XUhLgJiaiMgcGQcH",
		"book_2uxCPjmfJyirA09TOMXY5tmEjRF",
		"acht_2uxFp62TOOrxdeuaX6moTCVd4h8",
		"acht_2uxGEGquIvSlZG6Q2YYBMSQY45p",
		"acht_2uxHiqoazckG0uh3qWWyKHo1iFz",
		"rttr_2uxJSflLF2KX9xA6w4fMrXToqaX",
		"rttr_2uxMOfJAejxNLObXAcrjyiZEn6y",
		"wire_2uxPQx1Nlnk2fzt26EnMwtMWQzX",
		"wire_2uxQNfgfMUHh3s6shVIc4lLMou0",
		"wire_2uxQhPxF02K9p4A1ncmp69Q2tdv",
		"wire_2uxQugjSF5GZPKtcMMzAUfIyxth",
		"swft_2uxWg0rgJaGq5UXKNzbmZkL2LeJ",
		"swft_2uxX2d2mGihsVhaN69c5PEklMpk",
		"rttr_2uxyDpeSKnsBNEarE13herZsNzP",
		"book_2v4eOkIEtj0LYlk93oFnojHHxo3",
		"rttr_2v7ilPeQlH45EzVgw90EbvVm5RC",
		"swft_2v7knl3OlajJmyX7zRko8EwKndI",
		"swft_2v7lOSIAMBIwtPz9i3G2gL9YLiH",
		"swft_2v7s2T9nkbVVNbS97VsaUaAcxiC",
		"swft_2v7tvMguvO18Lm6gcq8gIm4DlP8",
		"swft_2v7voBvePIsdj88V8wjtDtqcxNU",
		"swft_2v7ydmhfZqZXD2U8IC5PqOhynzE",
		"swft_2v87g7JK4TCqsbi0cb5w57CHTT3",
		"swft_2v8HOm8Pppo9y0Cd3MofxYdi2NW",
		"book_2vE9ANaN27cbsFkTCQ9hOXd9ruk",
		"book_2vEGlzAXAqBUvnC1xMFpzN0DPsn",
		"book_2vEVIqeMJkSlzHeYkSGFZTn2sBJ",
		"book_2vEVULQlqR3iEoEonJ9UvpLDor2",
		"book_2vEWv8kvTyVwkuvn9GhIjeKBNgu",
		"book_2vEZKx2CNjlre9H6pd2IiReExQ5",
		"book_2vEaQpFKSpVnhXD9hmk5qE7xtEQ",
		"book_2vEbAaouaD3s4GKCpb3VDFAq1zF",
		"book_2vGSUC5dI3GbJBTjgqtWRCZG7BT",
		"book_2vGduCgXsoG2Bj6NOJCl6uBXXRD",
		"book_2vGeB1kOSZETP1zrQJ70rGK1VVC",
		"acht_2vj0cNx1iOorXp5vRRjfDnTPZWR",
		"swft_2wPMHtNu8bb6CKrvjfSL429fEF2",
		"swft_2wPQdve1bcPo2dzzfA1ihX5xp9K",
		"swft_2wPTzhf5QwNI8my3jrNhq7EGKnt",
		"swft_2wRSs9KNLSr4Qc2dA9ZMGdhET94",
		"swft_2wRn7Dm4fIMTQm0NtmyaoVXgWRA",
		"swft_2wipzrw8N7B6F6ksx8jxXG8ktc3",
		"swft_2wj2YHbyHLUxDBGr4JQNUcaBBKZ",
		"swft_2wjCEyjGTPcX7wHQwXsiQEGRkqy",
		"swft_2wjDgfx31X8yFRZYeepw6gmB4ho",
		"swft_2wrQFf0YvFgW3BV68dMeJNSHunD",
		"book_2xMLj5saheZbdwZdTspBZiZp52o",
		"book_2xMLpL9JYChOmOK6dyt89evcifl",
		"book_2xMNNJB05RhXuJ68MUKhEcJzsn4",
		"book_2xMNQDauuGKffGveePg9uJO7631",
		"book_2xMNWMbTVhbimbIC7IAJIWB7FZV",
		"book_2xMNYaYHX8ExuxjojMN19J9Fafe",
		"book_2xMNc1rcwekNqDxHB1bEYfiXysM",
		"book_2xMOEv3QgHuKiUU5NHfUUdjgOU8",
		"book_2xMOHfpAbGP11sdijpG0Ye19HOc",
		"book_2xMOJeChE3ObwZdGfi11k50VoDM",
		"book_2xMOMHIWczKXZalsiU84ds3fzeJ",
		"book_2xMONdzLI1HC8b3eZrbYlO1az6k",
		"book_2xMOQKFJ6a2m6cuy0HQpa5sfaip",
		"book_2xMOapDzZIQuSmKfX9xmTzTzgyu",
		"book_2xMOcs4GBfcySgtxtCXB1vZNqWz",
		"book_2xMQ0i6qIhencw6uW0jGjewT3TK",
		"book_2xMQ2TmjsBuG4QnX2McDSDmSL6r",
		"book_2xMQ5TAoZVYWR3rwse6XHADeiVX",
		"book_2xMQ7WpfOyoRCHkmH5ttYDvBkc6",
		"book_2xMQ9XcuS9ZSM0L1h64NBi4dBWs",
		"book_2xMQB6BBGm9pErzS31Y8gMgUslu",
		"book_2xMQJPaGtSfOxZqYHBKNDatpXPk",
		"book_2xMQKbiJofigbnJFuSPnGkyfvRx",
		"book_2xMQO3FTQSdvT6JvjbU1CSNj8Q5",
		"book_2xMQSlVbyKS69ig5xMXEBPTn6SO",
		"book_2xMQUhljarM3zXvmRrJj71L1TXM",
		"book_2xMQXMPE5BHeFlBGkioi9u63YBF",
		"book_2xMQamcZ5qKiney2ckiZophHuj9",
		"book_3FreUCBF9xpvYO8wnz7kyjFIJsL",
		"book_3FredDHtx2skzCGSiW1tfDWyoug",
		"book_3Frgn1Bdrn3NfIKAUi7Vx2u1e4W",
		"acht_3Frgn1rUp23Ot7HfBsxgyRButQ0",
		"book_3FtV54Lw9BpByriku3lRyN37ljX",
		"acht_3FtV4zWT6nxTzlvtmnwkn1CZVvg",
		"acht_3FtW4d9L34vJdEdSnXN6g47JExp",
		"book_3FtW4WiUy8bsqzYBTohVIInKt8z",
		"book_3FtWg54h93CmFkGFNwrkd3TXS5F",
		"acht_3FtWgCFjCfBupiWH1QxcUljUydB",
		"book_3FtX8AmWSmeogdZU8MbWvckpBdI",
		"acht_3FtX8HovWbvuLFyvLZOYRj7MK1a",
		"book_3FtXtg58QqWbMeHUo7WJJOyCWC8",
		"acht_3FtXtrXkEM1SFYPWoZv9UrChhVo",
		"book_3FtcdkfleACDGzNq7SuNqjP6kw7",
		"acht_3FtcdyuRVWwgnii8kQxLLnFkovr",

	}
)

// collectAllCounterpartyIDs pages through every counterparty and returns their
// IDs in list order.
func collectAllCounterpartyIDs(ctx context.Context, c Client) ([]string, error) {
	var ids []string
	cursor := ""
	for {
		counterparties, hasMore, err := c.GetCounterparties(ctx, cursor, contractPageSize)
		if err != nil {
			return nil, err
		}
		if len(counterparties) == 0 {
			break
		}
		for _, cp := range counterparties {
			ids = append(ids, cp.ID)
		}
		if !hasMore {
			break
		}
		cursor = counterparties[len(counterparties)-1].ID
	}
	return ids, nil
}

// collectAllTransactionIDs walks the client's Timeline the same way the connector
// does and returns every transaction ID in the chronological order the connector
// consumes. Bounded to avoid an accidental unbounded loop.
func collectAllTransactionIDs(ctx context.Context, c Client) ([]string, error) {
	var ids []string
	timeline := Timeline{}
	for i := 0; i < 5000; i++ {
		batch, tl, hasMore, err := c.GetTransactions(ctx, timeline, contractPageSize)
		if err != nil {
			return nil, err
		}
		timeline = tl
		for _, tx := range batch {
			ids = append(ids, tx.ID)
		}
		if !hasMore {
			break
		}
	}
	return ids, nil
}

var _ = Describe("Column API contract", func() {
	var (
		ctx context.Context
		c   Client
	)

	BeforeEach(func() {
		apiKey := os.Getenv("COLUMN_CONTRACT_API_KEY")
		if apiKey == "" {
			Skip("COLUMN_CONTRACT_API_KEY must be set to run the Column contract test")
		}

		ctx = context.Background()
		c = New("column", apiKey, contractEndpoint)
	})

	Describe("GetAccounts", func() {
		It("returns accounts whose shape matches what the connector consumes", func() {
			accounts, _, err := c.GetAccounts(ctx, "", contractPageSize)
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			ids := make([]string, 0, len(accounts))
			for _, a := range accounts {
				Expect(a.ID).ToNot(BeEmpty())
				Expect(a.Type).ToNot(BeEmpty())
				Expect(a.CurrencyCode).ToNot(BeEmpty())
				// The connector hard-parses created_at as RFC3339 (fillAccounts),
				// erroring otherwise — so it is a real contract field.
				_, perr := time.Parse(time.RFC3339, a.CreatedAt)
				Expect(perr).To(BeNil(), "account created_at %q is not RFC3339", a.CreatedAt)
				ids = append(ids, a.ID)
			}

			if contracttest.BootstrapEnabled("COLUMN") {
				contracttest.LogBootstrap("expectedAccountIDs", ids)
			}
		})

		It("returns accounts in the expected, stable order", func() {
			if len(expectedAccountIDs) == 0 {
				Skip("expectedAccountIDs is not populated — fill it from the bootstrap log to enable the ordering contract")
			}

			accounts, _, err := c.GetAccounts(ctx, "", contractPageSize)
			Expect(err).To(BeNil())

			gotIDs := make([]string, 0, len(accounts))
			for _, a := range accounts {
				gotIDs = append(gotIDs, a.ID)
			}
			Expect(gotIDs).To(Equal(expectedAccountIDs))
		})
	})

	Describe("GetAccountBalances", func() {
		It("returns the balance of an account by ID with numeric amount fields", func() {
			accounts, _, err := c.GetAccounts(ctx, "", contractPageSize)
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			balance, err := c.GetAccountBalances(ctx, accounts[0].ID)
			Expect(err).To(BeNil())
			Expect(balance).ToNot(BeNil())

			// The four amount fields are json.Number — they must be present and
			// parse as integers (Column amounts are integer minor units).
			contracttest.AssertIntegerAmount(balance.AvailableAmount, "available_amount")
			contracttest.AssertIntegerAmount(balance.HoldingAmount, "holding_amount")
			contracttest.AssertIntegerAmount(balance.LockedAmount, "locked_amount")
			contracttest.AssertIntegerAmount(balance.PendingAmount, "pending_amount")
		})
	})

	Describe("GetCounterparties", func() {
		It("returns counterparties whose shape matches what the connector consumes", func() {
			counterparties, _, err := c.GetCounterparties(ctx, "", contractPageSize)
			Expect(err).To(BeNil())
			Expect(counterparties).ToNot(BeEmpty())

			for _, cp := range counterparties {
				Expect(cp.ID).ToNot(BeEmpty())
				// Name is optional (the connector tolerates an empty name); the
				// hard dependency is created_at, which fillExternalAccounts parses
				// as RFC3339 and errors on otherwise.
				_, perr := time.Parse(time.RFC3339, cp.CreatedAt)
				Expect(perr).To(BeNil(), "counterparty created_at %q is not RFC3339", cp.CreatedAt)
			}

			if contracttest.BootstrapEnabled("COLUMN") {
				allIDs, err := collectAllCounterpartyIDs(ctx, c)
				Expect(err).To(BeNil())
				contracttest.LogBootstrap("expectedCounterpartyIDs", allIDs)
			}
		})

		It("keeps the known counterparties in their expected, stable relative order", func() {
			if len(expectedCounterpartyIDs) == 0 {
				Skip("expectedCounterpartyIDs is not populated — fill it from the bootstrap log to enable the ordering contract")
			}

			// This suite creates a new counterparty on every run and Column has no
			// delete, so the live list grows and new counterparties are prepended.
			// Assert only that the *pinned* counterparties still appear in the same
			// relative order — filtering out any others — rather than requiring an
			// exact full-list match. Page through all counterparties because, as
			// the list grows, the pinned (older) IDs eventually spill past page 1.
			allIDs, err := collectAllCounterpartyIDs(ctx, c)
			Expect(err).To(BeNil())
			gotKnownIDs := contracttest.FilterToPinned(allIDs, expectedCounterpartyIDs)
			Expect(gotKnownIDs).To(Equal(expectedCounterpartyIDs))
		})
	})

	Describe("GetTransactions", func() {
		It("returns transactions whose shape matches what the connector consumes", func() {
			// A fresh Timeline drives the client's backlog scan; we only assert
			// schema, not ordering (the timeline scan + live settlement make
			// ordering volatile).
			transactions, _, _, err := c.GetTransactions(ctx, Timeline{}, contractPageSize)
			Expect(err).To(BeNil())

			for _, tx := range transactions {
				Expect(tx.ID).ToNot(BeEmpty())
				Expect(tx.Status).ToNot(BeEmpty())
				Expect(tx.Type).ToNot(BeEmpty())
				Expect(tx.CurrencyCode).ToNot(BeEmpty())
			}

			if contracttest.BootstrapEnabled("COLUMN") {
				allIDs, err := collectAllTransactionIDs(ctx, c)
				Expect(err).To(BeNil())
				contracttest.LogBootstrap("expectedTransactionIDs", allIDs)
			}
		})

		It("keeps the known transactions in their expected, stable relative order", func() {
			if len(expectedTransactionIDs) == 0 {
				Skip("expectedTransactionIDs is not populated — set COLUMN_CONTRACT_BOOTSTRAP=1 and fill it from the bootstrap log to enable the ordering contract")
			}

			// The suite creates new transactions on every run (book transfer + ACH
			// payout), which the Timeline walk returns at the chronological end.
			// Assert only that the *pinned* transactions retain their relative
			// order, ignoring any newly created ones.
			allIDs, err := collectAllTransactionIDs(ctx, c)
			Expect(err).To(BeNil())
			gotKnownIDs := contracttest.FilterToPinned(allIDs, expectedTransactionIDs)
			Expect(gotKnownIDs).To(Equal(expectedTransactionIDs))
		})
	})

	Describe("ListEventSubscriptions", func() {
		It("returns event subscriptions whose shape matches what the connector consumes", func() {
			subscriptions, err := c.ListEventSubscriptions(ctx)
			Expect(err).To(BeNil())

			for _, s := range subscriptions {
				Expect(s.ID).ToNot(BeEmpty())
				Expect(s.URL).ToNot(BeEmpty())
			}
		})
	})

	Describe("event subscription lifecycle", func() {
		It("creates a webhook endpoint with a valid, retrievable shape, then deletes it", func() {
			created, err := c.CreateEventSubscription(ctx, &CreateEventSubscriptionRequest{
				URL:           contractEventSubscriptionURL,
				EnabledEvents: []string{string(EventCategoryBookTransferCompleted)},
			})
			Expect(err).To(BeNil())
			Expect(created).ToNot(BeNil())
			Expect(created.ID).ToNot(BeEmpty())

			DeferCleanup(func() {
				_, derr := c.DeleteEventSubscription(ctx, created.ID)
				Expect(derr).To(BeNil())
			})

			Expect(created.URL).To(Equal(contractEventSubscriptionURL))
			Expect(created.Secret).ToNot(BeEmpty())
			Expect(created.EnabledEvents).To(ContainElement(string(EventCategoryBookTransferCompleted)))
		})
	})

	Describe("CreateCounterPartyBankAccount", func() {
		// Column has no delete-counterparty method, so each run accumulates one
		// counterparty in the sandbox. This is accepted (small, sandbox-only).
		It("creates a counterparty whose shape matches what the connector consumes", func() {
			resp, err := c.CreateCounterPartyBankAccount(ctx, CounterPartyBankAccountRequest{
				Name:              "Formance Contract Test",
				RoutingNumber:     "121000248", // checksum-valid ABA (Wells Fargo)
				AccountNumber:     "1234567890",
				AccountType:       "checking",
				RoutingNumberType: "aba",
			})
			Expect(err).To(BeNil())
			Expect(resp.ID).ToNot(BeEmpty())
			Expect(resp.Name).ToNot(BeEmpty())
			Expect(resp.RoutingNumber).ToNot(BeEmpty())
		})
	})

	Describe("InitiateTransfer", func() {
		// Internal book transfer between two of our own accounts: money stays on
		// the platform. Minimal amount, AllowOverdraft so a zero-balance sandbox
		// account doesn't fail, unique reference per run.
		It("initiates a minimal internal book transfer", func() {
			accounts, _, err := c.GetAccounts(ctx, "", contractPageSize)
			Expect(err).To(BeNil())
			if len(accounts) < 2 {
				Skip("need at least 2 internal accounts in the sandbox to exercise a book transfer")
			}

			resp, err := c.InitiateTransfer(ctx, &TransferRequest{
				Amount:                1,
				CurrencyCode:          "USD",
				SenderBankAccountId:   accounts[0].ID,
				ReceiverBankAccountId: accounts[1].ID,
				AllowOverdraft:        true,
				Details: TransferRequestDetails{
					SenderName: "Formance Contract Test",
				},
				Reference: contracttest.Ref("column", "book"),
			})
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ID).ToNot(BeEmpty())
			Expect(resp.Status).ToNot(BeEmpty())
			Expect(resp.Amount).To(Equal(int64(1)))
			Expect(resp.CurrencyCode).ToNot(BeEmpty())
		})
	})

	Describe("InitiatePayout (ACH)", func() {
		// Outbound ACH payout to a counterparty. Minimal amount, AllowOverdraft,
		// unique reference per run. Source/destination derived from the reads.
		It("initiates a minimal ACH payout", func() {
			accounts, _, err := c.GetAccounts(ctx, "", contractPageSize)
			Expect(err).To(BeNil())
			Expect(accounts).ToNot(BeEmpty())

			counterparties, _, err := c.GetCounterparties(ctx, "", contractPageSize)
			Expect(err).To(BeNil())
			if len(counterparties) == 0 {
				Skip("need at least 1 counterparty in the sandbox to exercise an ACH payout")
			}

			resp, err := c.InitiatePayout(ctx, &PayoutRequest{
				Amount:             1,
				CurrencyCode:       "USD",
				SourceAccount:      accounts[0].ID,
				DestinationAccount: counterparties[0].ID,
				Description:        "Formance Contract Test",
				Reference:          contracttest.Ref("column", "ach"),
				Metadata: map[string]string{
					ColumnPayoutTypeMetadataKey:     "ach",
					ColumnTypeMetadataKey:           "CREDIT", // Column ACH type enum is uppercase CREDIT/DEBIT
					ColumnEntryClassCodeMetadataKey: "PPD",
					ColumnAllowOverdraftMetadataKey: "true",
				},
			})
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ID).ToNot(BeEmpty())
			Expect(resp.Status).ToNot(BeEmpty())
		})
	})
})
