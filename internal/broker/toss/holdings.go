package toss

import (
	"context"
	"fmt"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/shopspring/decimal"
)

const holdingsPath = "/api/v1/holdings"

// priceDTO is the Toss per-currency aggregate ("Price"): krw is always present,
// usd is null when there are no USD-denominated holdings.
type priceDTO struct {
	KRW string  `json:"krw"`
	USD *string `json:"usd"`
}

type holdingsItemDTO struct {
	Symbol               string `json:"symbol"`
	Name                 string `json:"name"`
	MarketCountry        string `json:"marketCountry"`
	Currency             string `json:"currency"`
	Quantity             string `json:"quantity"`
	LastPrice            string `json:"lastPrice"`
	AveragePurchasePrice string `json:"averagePurchasePrice"`
	MarketValue          struct {
		Amount string `json:"amount"`
	} `json:"marketValue"`
	ProfitLoss struct {
		Amount string `json:"amount"`
		Rate   string `json:"rate"`
	} `json:"profitLoss"`
	DailyProfitLoss struct {
		Amount string `json:"amount"`
		Rate   string `json:"rate"`
	} `json:"dailyProfitLoss"`
}

type holdingsOverviewDTO struct {
	TotalPurchaseAmount priceDTO `json:"totalPurchaseAmount"`
	MarketValue         struct {
		Amount priceDTO `json:"amount"`
	} `json:"marketValue"`
	ProfitLoss struct {
		Amount priceDTO `json:"amount"`
	} `json:"profitLoss"`
	DailyProfitLoss struct {
		Amount priceDTO `json:"amount"`
	} `json:"dailyProfitLoss"`
	Items []holdingsItemDTO `json:"items"`
}

// Holdings implements broker.Broker.
func (c *Client) Holdings(ctx context.Context, acct domain.Account) (domain.HoldingsOverview, error) {
	dto, err := getJSON[holdingsOverviewDTO](ctx, c, holdingsPath, nil, accountHeaders(acct))
	if err != nil {
		return domain.HoldingsOverview{}, err
	}
	return dto.toDomain()
}

func (o holdingsOverviewDTO) toDomain() (domain.HoldingsOverview, error) {
	var out domain.HoldingsOverview

	totals, err := o.currencyTotals()
	if err != nil {
		return domain.HoldingsOverview{}, err
	}
	out.Totals = totals

	out.Positions = make([]domain.Position, 0, len(o.Items))
	for i := range o.Items {
		pos, err := o.Items[i].toDomain()
		if err != nil {
			return domain.HoldingsOverview{}, err
		}
		out.Positions = append(out.Positions, pos)
	}
	return out, nil
}

// currencyTotals builds one CurrencyTotals per currency present in the overview.
// KRW is always present; USD only when it has a value.
func (o holdingsOverviewDTO) currencyTotals() ([]domain.CurrencyTotals, error) {
	totals := make([]domain.CurrencyTotals, 0, 2)

	krw, err := o.totalsFor(domain.KRW)
	if err != nil {
		return nil, err
	}
	totals = append(totals, krw)

	if o.TotalPurchaseAmount.USD != nil {
		usd, err := o.totalsFor(domain.USD)
		if err != nil {
			return nil, err
		}
		totals = append(totals, usd)
	}
	return totals, nil
}

func (o holdingsOverviewDTO) totalsFor(cur domain.Currency) (domain.CurrencyTotals, error) {
	purchase, err := moneyFromPrice(o.TotalPurchaseAmount, cur)
	if err != nil {
		return domain.CurrencyTotals{}, err
	}
	marketValue, err := moneyFromPrice(o.MarketValue.Amount, cur)
	if err != nil {
		return domain.CurrencyTotals{}, err
	}
	totalPnL, err := moneyFromPrice(o.ProfitLoss.Amount, cur)
	if err != nil {
		return domain.CurrencyTotals{}, err
	}
	dailyPnL, err := moneyFromPrice(o.DailyProfitLoss.Amount, cur)
	if err != nil {
		return domain.CurrencyTotals{}, err
	}

	// The overview reports a single blended (KRW-converted) rate, not a
	// per-currency one, so derive each currency's total return from amounts.
	var totalRate decimal.Decimal
	if !purchase.Amount.IsZero() {
		totalRate = totalPnL.Amount.Div(purchase.Amount)
	}

	return domain.CurrencyTotals{
		Currency:       cur,
		PurchaseAmount: purchase,
		MarketValue:    marketValue,
		PnL: domain.PnLSet{
			Total: domain.PnL{Amount: totalPnL, Rate: totalRate},
			Daily: domain.PnL{Amount: dailyPnL},
		},
	}, nil
}

func (it holdingsItemDTO) toDomain() (domain.Position, error) {
	cur := domain.Currency(it.Currency)

	quantity, err := decimal.NewFromString(it.Quantity)
	if err != nil {
		return domain.Position{}, fmt.Errorf("holdings: parse quantity %q: %w", it.Quantity, err)
	}
	avgCost, err := domain.ParseMoney(it.AveragePurchasePrice, cur)
	if err != nil {
		return domain.Position{}, err
	}
	lastPrice, err := domain.ParseMoney(it.LastPrice, cur)
	if err != nil {
		return domain.Position{}, err
	}
	marketValue, err := domain.ParseMoney(it.MarketValue.Amount, cur)
	if err != nil {
		return domain.Position{}, err
	}

	totalPnL, err := pnlFrom(it.ProfitLoss.Amount, it.ProfitLoss.Rate, cur)
	if err != nil {
		return domain.Position{}, err
	}
	dailyPnL, err := pnlFrom(it.DailyProfitLoss.Amount, it.DailyProfitLoss.Rate, cur)
	if err != nil {
		return domain.Position{}, err
	}

	return domain.Position{
		Symbol:      it.Symbol,
		Name:        it.Name,
		Market:      domain.Market(it.MarketCountry),
		Currency:    cur,
		Quantity:    quantity,
		AvgCost:     avgCost,
		LastPrice:   lastPrice,
		MarketValue: marketValue,
		PnL:         domain.PnLSet{Total: totalPnL, Daily: dailyPnL},
	}, nil
}

// moneyFromPrice extracts the amount for cur from a per-currency Price. A null
// USD field is treated as zero.
func moneyFromPrice(p priceDTO, cur domain.Currency) (domain.Money, error) {
	switch cur {
	case domain.KRW:
		return domain.ParseMoney(p.KRW, cur)
	case domain.USD:
		if p.USD == nil {
			return domain.NewMoney(decimal.Zero, cur), nil
		}
		return domain.ParseMoney(*p.USD, cur)
	default:
		return domain.Money{}, fmt.Errorf("holdings: unsupported currency %q", cur)
	}
}

func pnlFrom(amount, rate string, cur domain.Currency) (domain.PnL, error) {
	m, err := domain.ParseMoney(amount, cur)
	if err != nil {
		return domain.PnL{}, err
	}
	r, err := decimal.NewFromString(rate)
	if err != nil {
		return domain.PnL{}, fmt.Errorf("holdings: parse rate %q: %w", rate, err)
	}
	return domain.PnL{Amount: m, Rate: r}, nil
}
