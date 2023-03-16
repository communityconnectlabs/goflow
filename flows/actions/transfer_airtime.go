package actions

import (
	"github.com/nyaruka/gocommon/urns"
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/events"
	"github.com/pkg/errors"

	"github.com/shopspring/decimal"
)

func init() {
	registerType(TypeTransferAirtime, func() flows.Action { return &TransferAirtimeAction{} })
}

var transferCategories = []string{CategorySuccess, CategoryFailure}

// TypeTransferAirtime is the type for the transfer airtime action
const TypeTransferAirtime string = "transfer_airtime"

// TransferAirtimeAction attempts to make an airtime transfer to the contact.
//
// An [event:airtime_transferred] event will be created if the airtime could be sent.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "transfer_airtime",
//     "amounts": {"RWF": 500, "USD": 0.5},
//     "result_name": "Reward Transfer"
//   }
//
// @action transfer_airtime
type TransferAirtimeAction struct {
	baseAction
	onlineAction

	Amounts    map[string]decimal.Decimal `json:"amounts" validate:"required"`
	ResultName string                     `json:"result_name" validate:"required"`
}

// NewTransferAirtime creates a new airtime transfer action
func NewTransferAirtime(uuid flows.ActionUUID, amounts map[string]decimal.Decimal, resultName string) *TransferAirtimeAction {
	return &TransferAirtimeAction{
		baseAction: newBaseAction(TypeTransferAirtime, uuid),
		Amounts:    amounts,
		ResultName: resultName,
	}
}

// Execute executes the transfer action
func (a *TransferAirtimeAction) Execute(run flows.Run, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	transfer, err := a.transfer(run, step, logEvent)
	if err != nil {
		logEvent(events.NewError(err))

		a.saveFailure(run, step, logEvent)
	} else {
		a.saveSuccess(run, step, transfer, logEvent)
	}

	return nil
}

func (a *TransferAirtimeAction) transfer(run flows.Run, step flows.Step, logEvent flows.EventCallback) (*flows.AirtimeTransfer, error) {
	// fail if we don't have a contact
	contact := run.Contact()
	if contact == nil {
		return nil, errors.New("can't execute action in session without a contact")
	}

	// fail if the contact doesn't have a tel URN
	telURNs := contact.URNs().WithScheme(urns.TelScheme)
	if len(telURNs) == 0 {
		return nil, errors.New("can't transfer airtime to contact without a tel URN")
	}

	// if contact's preferred channel is tel, use that as the sender
	var sender urns.URN
	channel := contact.PreferredChannel()
	if channel != nil && channel.SupportsScheme(urns.TelScheme) {
		sender, _ = urns.Parse("tel:" + channel.Address())
	}

	svc, err := run.Session().Engine().Services().Airtime(run.Session())
	if err != nil {
		return nil, err
	}

	httpLogger := &flows.HTTPLogger{}

	transfer, err := svc.Transfer(run.Session(), sender, telURNs[0].URN(), a.Amounts, httpLogger.Log)
	if transfer != nil {
		logEvent(events.NewAirtimeTransferred(transfer, httpLogger.Logs))
	}

	return transfer, err
}

func (a *TransferAirtimeAction) saveSuccess(run flows.Run, step flows.Step, transfer *flows.AirtimeTransfer, logEvent flows.EventCallback) {
	a.saveResult(run, step, a.ResultName, transfer.ActualAmount.String(), CategorySuccess, "", "", nil, logEvent)
}

func (a *TransferAirtimeAction) saveFailure(run flows.Run, step flows.Step, logEvent flows.EventCallback) {
	a.saveResult(run, step, a.ResultName, "0", CategoryFailure, "", "", nil, logEvent)
}

// Results enumerates any results generated by this flow object
func (a *TransferAirtimeAction) Results(include func(*flows.ResultInfo)) {
	if a.ResultName != "" {
		include(flows.NewResultInfo(a.ResultName, transferCategories))
	}
}
