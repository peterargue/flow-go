package data_providers

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/access"
	commonmodels "github.com/onflow/flow-go/engine/access/rest/common/models"
	"github.com/onflow/flow-go/engine/access/rest/common/parser"
	"github.com/onflow/flow-go/engine/access/rest/websockets/models"
	"github.com/onflow/flow-go/engine/access/subscription"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/counters"

	"github.com/onflow/flow/protobuf/go/flow/entities"
)

// transactionStatusesArguments contains the arguments required for subscribing to transaction statuses
type transactionStatusesArguments struct {
	TxID flow.Identifier `json:"tx_id"` // ID of the transaction to monitor.
}

// TransactionStatusesDataProvider is responsible for providing tx statuses
type TransactionStatusesDataProvider struct {
	*baseDataProvider

	arguments     transactionStatusesArguments
	messageIndex  counters.StrictMonotonicCounter
	linkGenerator commonmodels.LinkGenerator
}

var _ DataProvider = (*TransactionStatusesDataProvider)(nil)

func NewTransactionStatusesDataProvider(
	logger zerolog.Logger,
	api access.API,
	subscriptionID string,
	linkGenerator commonmodels.LinkGenerator,
	topic string,
	rawArguments models.Arguments,
	send chan<- interface{},
) (*TransactionStatusesDataProvider, error) {
	args, err := parseTransactionStatusesArguments(rawArguments)
	if err != nil {
		return nil, fmt.Errorf("invalid arguments for tx statuses data provider: %w", err)
	}
	provider := newBaseDataProvider(
		logger.With().Str("component", "transaction-statuses-data-provider").Logger(),
		api,
		subscriptionID,
		topic,
		rawArguments,
		send,
	)

	return &TransactionStatusesDataProvider{
		baseDataProvider: provider,
		arguments:        args,
		messageIndex:     counters.NewMonotonicCounter(0),
		linkGenerator:    linkGenerator,
	}, nil
}

// Run starts processing the subscription for events and handles responses.
//
// No errors are expected during normal operations.
func (p *TransactionStatusesDataProvider) Run(ctx context.Context) error {
	// start a new subscription. we read data from it and send them to client's channel
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	p.subscriptionState = newSubscriptionState(cancel, p.createAndStartSubscription(ctx, p.arguments))

	// set messageIndex to zero in case Run() called for the second time
	p.messageIndex = counters.NewMonotonicCounter(0)

	return run(
		p.baseDataProvider.done,
		p.subscriptionState.subscription,
		func(response []*access.TransactionResult) error {
			return p.sendResponse(response, &p.messageIndex)
		},
	)
}

// sendResponse processes a tx status message and sends it to client's channel.
// This function is not expected to be called concurrently.
//
// No errors are expected during normal operations.
func (p *TransactionStatusesDataProvider) sendResponse(
	txResults []*access.TransactionResult,
	messageIndex *counters.StrictMonotonicCounter,
) error {
	for i := range txResults {
		var txStatusesPayload models.TransactionStatusesResponse
		txStatusesPayload.Build(p.linkGenerator, txResults[i], messageIndex.Value())

		var response models.BaseDataProvidersResponse
		response.Build(p.ID(), p.Topic(), &txStatusesPayload)

		messageIndex.Increment()
		p.send <- &response
	}

	return nil
}

// createAndStartSubscription creates a new subscription using the specified input arguments.
func (p *TransactionStatusesDataProvider) createAndStartSubscription(
	ctx context.Context,
	args transactionStatusesArguments,
) subscription.Subscription {
	return p.api.SubscribeTransactionStatuses(ctx, args.TxID, entities.EventEncodingVersion_JSON_CDC_V0)
}

// parseAccountStatusesArguments validates and initializes the account statuses arguments.
func parseTransactionStatusesArguments(
	arguments models.Arguments,
) (transactionStatusesArguments, error) {
	allowedFields := map[string]struct{}{
		"tx_id": {},
	}
	err := ensureAllowedFields(arguments, allowedFields)
	if err != nil {
		return transactionStatusesArguments{}, err
	}

	var args transactionStatusesArguments

	// Check if tx_id exists and is not empty
	rawTxID, exists := arguments["tx_id"]
	if !exists {
		return transactionStatusesArguments{}, fmt.Errorf("missing 'tx_id' field")
	}

	// Ensure the transaction ID is a string
	txIDString, isString := rawTxID.(string)
	if !isString {
		return transactionStatusesArguments{}, fmt.Errorf("'tx_id' must be a string")
	}

	if len(txIDString) == 0 {
		return transactionStatusesArguments{}, fmt.Errorf("'tx_id' must not be empty")
	}

	var parsedTxID parser.ID
	if err = parsedTxID.Parse(txIDString); err != nil {
		return transactionStatusesArguments{}, fmt.Errorf("invalid 'tx_id': %w", err)
	}

	// Assign the validated transaction ID to the args
	args.TxID = parsedTxID.Flow()
	return args, nil
}
