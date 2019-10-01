package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/assets/static"
	_ "github.com/greatnonprofits-nfp/goflow/extensions/transferto"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/engine"
	"github.com/greatnonprofits-nfp/goflow/flows/events"
	"github.com/greatnonprofits-nfp/goflow/flows/resumes"
	"github.com/greatnonprofits-nfp/goflow/flows/triggers"
	"github.com/greatnonprofits-nfp/goflow/utils"

	"github.com/pkg/errors"
)

var contactJSON = `{
	"uuid": "ba96bf7f-bc2a-4873-a7c7-254d1927c4e3",
	"id": 1234567,
	"name": "Ben Haggerty",
	"created_on": "2018-01-01T12:00:00.000000000-00:00",
	"fields": {},
	"timezone": "America/Guayaquil",
	"urns": [
		"tel:+12065551212",
		"facebook:1122334455667788",
		"mailto:ben@macklemore"
	]
}
`

var usage = `usage: flowrunner [flags] <assets.json> <flow_uuid>`

func main() {
	var initialMsg, contactLang string
	var printRepro bool
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.StringVar(&initialMsg, "msg", "", "initial message to trigger session with")
	flags.StringVar(&contactLang, "lang", "eng", "initial language of the contact")
	flags.BoolVar(&printRepro, "repro", false, "print repro afterwards")
	flags.Parse(os.Args[1:])
	args := flags.Args()

	if len(args) != 2 {
		fmt.Println(usage)
		flags.PrintDefaults()
		os.Exit(1)
	}

	assetsPath := args[0]
	flowUUID := assets.FlowUUID(args[1])

	repro, err := RunFlow(assetsPath, flowUUID, initialMsg, utils.Language(contactLang), os.Stdin, os.Stdout)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if printRepro {
		fmt.Println("---------------------------------------")
		marshaledRepro, _ := utils.JSONMarshalPretty(repro)
		fmt.Println(string(marshaledRepro))
	}
}

// RunFlow steps through a flow
func RunFlow(assetsPath string, flowUUID assets.FlowUUID, initialMsg string, contactLang utils.Language, in io.Reader, out io.Writer) (*Repro, error) {
	source, err := static.LoadSource(assetsPath)
	if err != nil {
		return nil, err
	}

	sa, err := engine.NewSessionAssets(source)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing assets")
	}

	flow, err := sa.Flows().Get(flowUUID)
	if err != nil {
		return nil, err
	}

	if err := flow.ValidateRecursive(sa, nil); err != nil {
		return nil, err
	}

	contact, err := flows.ReadContact(sa, json.RawMessage(contactJSON), assets.PanicOnMissing)
	if err != nil {
		return nil, err
	}
	contact.SetLanguage(contactLang)

	// create our environment
	la, _ := time.LoadLocation("America/Los_Angeles")
	languages := []utils.Language{flow.Language(), contact.Language()}
	env := utils.NewEnvironmentBuilder().WithTimezone(la).WithAllowedLanguages(languages).Build()

	repro := &Repro{}

	if initialMsg != "" {
		msg := createMessage(contact, initialMsg)
		repro.Trigger = triggers.NewMsgTrigger(env, flow.Reference(), contact, msg, nil)
	} else {
		repro.Trigger = triggers.NewManualTrigger(env, flow.Reference(), contact, nil)
	}
	fmt.Fprintf(out, "Starting flow '%s'....\n---------------------------------------\n", flow.Name())

	eng := engine.NewBuilder().WithDefaultUserAgent("goflow-flowrunner").Build()

	// start our session
	session, sprint, err := eng.NewSession(sa, repro.Trigger)
	if err != nil {
		return nil, err
	}

	printEvents(sprint.Events(), out)
	scanner := bufio.NewScanner(in)

	for session.Wait() != nil {

		// ask for input
		fmt.Fprintf(out, "> ")
		scanner.Scan()

		text := scanner.Text()
		var resume flows.Resume

		// create our resume
		if text == "/timeout" {
			resume = resumes.NewWaitTimeoutResume(nil, nil)
		} else {
			msg := createMessage(contact, scanner.Text())
			resume = resumes.NewMsgResume(nil, nil, msg)
		}

		repro.Resumes = append(repro.Resumes, resume)

		sprint, err := session.Resume(resume)
		if err != nil {
			return nil, err
		}

		printEvents(sprint.Events(), out)
	}

	return repro, nil
}

func createMessage(contact *flows.Contact, text string) *flows.MsgIn {
	return flows.NewMsgIn(flows.MsgUUID(utils.NewUUID()), contact.URNs()[0].URN(), nil, text, []utils.Attachment{})
}

func printEvents(log []flows.Event, out io.Writer) {
	for _, event := range log {
		var msg string
		switch typed := event.(type) {
		case *events.BroadcastCreatedEvent:
			text := typed.Translations[typed.BaseLanguage].Text
			msg = fmt.Sprintf("🔉 broadcasted '%s' to ...", text)
		case *events.ContactFieldChangedEvent:
			var action string
			if typed.Value != nil {
				action = fmt.Sprintf("changed to '%s'", typed.Value.Text)
			} else {
				action = "cleared"
			}
			msg = fmt.Sprintf("✏️ field '%s' %s", typed.Field.Key, action)
		case *events.ContactGroupsChangedEvent:
			msgs := make([]string, 0)
			if len(typed.GroupsAdded) > 0 {
				groups := make([]string, len(typed.GroupsAdded))
				for i, group := range typed.GroupsAdded {
					groups[i] = fmt.Sprintf("'%s'", group.Name)
				}
				msgs = append(msgs, "added to "+strings.Join(groups, ", "))
			}
			if len(typed.GroupsRemoved) > 0 {
				groups := make([]string, len(typed.GroupsRemoved))
				for i, group := range typed.GroupsRemoved {
					groups[i] = fmt.Sprintf("'%s'", group.Name)
				}
				msgs = append(msgs, "removed from "+strings.Join(groups, ", "))
			}
			msg = fmt.Sprintf("👪 %s", strings.Join(msgs, ", "))
		case *events.ContactLanguageChangedEvent:
			msg = fmt.Sprintf("🌐 language changed to '%s'", typed.Language)
		case *events.ContactNameChangedEvent:
			msg = fmt.Sprintf("📛 name changed to '%s'", typed.Name)
		case *events.ContactRefreshedEvent:
			msg = "👤 contact refreshed on resume"
		case *events.ContactTimezoneChangedEvent:
			msg = fmt.Sprintf("🕑 timezone changed to '%s'", typed.Timezone)
		case *events.EnvironmentRefreshedEvent:
			msg = "⚙️ environment refreshed on resume"
		case *events.ErrorEvent:
			msg = fmt.Sprintf("⚠️ %s", typed.Text)
		case *events.FlowEnteredEvent:
			msg = fmt.Sprintf("↪️ entered flow '%s'", typed.Flow.Name)
		case *events.InputLabelsAddedEvent:
			labels := make([]string, len(typed.Labels))
			for i, label := range typed.Labels {
				labels[i] = fmt.Sprintf("'%s'", label.Name)
			}
			msg = fmt.Sprintf("🏷️ labeled with %s", strings.Join(labels, ", "))
		case *events.IVRCreatedEvent:
			msg = fmt.Sprintf("📞 IVR created \"%s\"", typed.Msg.Text())
		case *events.MsgCreatedEvent:
			msg = fmt.Sprintf("💬 message created \"%s\"", typed.Msg.Text())
		case *events.MsgReceivedEvent:
			msg = fmt.Sprintf("📥 message received \"%s\"", typed.Msg.Text())
		case *events.MsgWaitEvent:
			if typed.TimeoutSeconds != nil {
				msg = fmt.Sprintf("⏳ waiting for message (%d sec timeout, type /timeout to simulate)....", *typed.TimeoutSeconds)
			} else {
				msg = "⏳ waiting for message...."
			}
		case *events.RunExpiredEvent:
			msg = "📆 exiting due to expiration"
		case *events.RunResultChangedEvent:
			msg = fmt.Sprintf("📈 run result '%s' changed to '%s'", typed.Name, typed.Value)
		case *events.SessionTriggeredEvent:
			msg = fmt.Sprintf("🏁 session triggered for '%s'", typed.Flow.Name)
		case *events.WaitTimedOutEvent:
			msg = "⏲️ resuming due to wait timeout"
		case *events.WebhookCalledEvent:
			url := truncate(typed.URL, 50)
			msg = fmt.Sprintf("☁️ called %s", url)
		default:
			msg = fmt.Sprintf("❓ %s event", typed.Type())
		}

		fmt.Fprintln(out, msg)
	}
}

// Repro describes the trigger and resumes needed to reproduce this session
type Repro struct {
	Trigger flows.Trigger  `json:"trigger"`
	Resumes []flows.Resume `json:"resumes"`
}

func truncate(str string, length int) string {
	ending := "..."
	runes := []rune(str)
	if len(runes) > length {
		return string(runes[0:length-len(ending)]) + ending
	}
	return str
}
