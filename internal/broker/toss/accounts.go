package toss

import (
	"context"

	"github.com/byunjuneseok/s8s/internal/domain"
)

const accountsPath = "/api/v1/accounts"

type accountDTO struct {
	AccountNo   string `json:"accountNo"`
	AccountSeq  int64  `json:"accountSeq"`
	AccountType string `json:"accountType"`
}

func (d accountDTO) toDomain() domain.Account {
	return domain.Account{
		No:   d.AccountNo,
		Seq:  d.AccountSeq,
		Type: domain.AccountType(d.AccountType),
	}
}

// Accounts implements broker.Broker.
func (c *Client) Accounts(ctx context.Context) ([]domain.Account, error) {
	dtos, err := getJSON[[]accountDTO](ctx, c, accountsPath, nil, nil)
	if err != nil {
		return nil, err
	}
	accounts := make([]domain.Account, len(dtos))
	for i, d := range dtos {
		accounts[i] = d.toDomain()
	}
	return accounts, nil
}
