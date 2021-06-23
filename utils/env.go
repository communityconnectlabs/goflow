package utils

import "os"

const (
	XParseApplicationId     = "X-Parse-Application-Id"
	XParseMasterKey         = "X-Parse-Master-Key"
	ParseAppId              = "MAILROOM_PARSE_SERVER_APP_ID"
	ParseMasterKey          = "MAILROOM_PARSE_SERVER_MASTER_KEY"
	ParseServerUrl          = "MAILROOM_PARSE_SERVER_URL"
	GiftcardCheckType       = "GIFTCARD_CHECK"
	ShortenURLPing          = "MAILROOM_SHORTEN_URL_PING"
	YoURLsHost              = "MAILROOM_YOURLS_HOST"
	YoURLsLogin             = "MAILROOM_YOURLS_LOGIN"
	YoURLsPassword          = "MAILROOM_YOURLS_PASSWORD"
	MailroomDomain          = "MAILROOM_DOMAIN"
	MailroomSpellCheckerKey = "MAILROOM_SPELL_CHECKER_KEY"
)

// Get environment variables passing a default value
func GetEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
