package smtpx_test

import (
	"testing"

	"github.com/greatnonprofits-nfp/goflow/utils/smtpx"

	"github.com/stretchr/testify/assert"
)

func TestMockSender(t *testing.T) {
	defer smtpx.SetSender(smtpx.DefaultSender)

	// a sender which succeeds
	sender := smtpx.NewMockSender("")
	smtpx.SetSender(sender)

	err := smtpx.Send("mail.temba.io", 255, "leah", "pass123", "updates@temba.io", []string{"bob@nyaruka.com", "jim@nyaruka.com"}, "Updates", "Have a great week", nil)
	assert.NoError(t, err)
	err = smtpx.Send("mail.temba.io", 255, "leah", "pass123", "updates@temba.io", []string{"bob@nyaruka.com", "jim@nyaruka.com"}, "Updates", "Have a great weekend", nil)
	assert.NoError(t, err)

	assert.Equal(t, []string{
		"HELO localhost\nMAIL FROM:<updates@temba.io>\nRCPT TO:<bob@nyaruka.com>\nRCPT TO:<jim@nyaruka.com>\nDATA\nHave a great week\n.\nQUIT\n",
		"HELO localhost\nMAIL FROM:<updates@temba.io>\nRCPT TO:<bob@nyaruka.com>\nRCPT TO:<jim@nyaruka.com>\nDATA\nHave a great weekend\n.\nQUIT\n",
	}, sender.Logs())

	// a sender which errors
	sender = smtpx.NewMockSender("oops can't send")
	smtpx.SetSender(sender)

	err = smtpx.Send("mail.temba.io", 25, "leah", "pass123", "updates@temba.io", []string{"bob@nyaruka.com", "jim@nyaruka.com"}, "Updates", "Have a great week", nil)

	assert.EqualError(t, err, "oops can't send")
	assert.Equal(t, 0, len(sender.Logs()))
}
