package config

type Config struct {
	Port         int
	Domain       string
	Development  bool
	Telegraph    Telegraph
	XRay         XRay
	Hugo         Hugo
	Twitter      Twitter
	Telegram     Telegram
	BunnyCDN     BunnyCDN
	WebmentionIO WebmentionIO
	Webhook      Webhook
	IndieAuth    IndieAuth
}

type Twitter struct {
	Key         string
	Secret      string
	Token       string
	TokenSecret string `mapstructure:"token_secret"`
}

type Telegram struct {
	Token  string
	ChatID int64 `mapstructure:"chat_id"`
}

type BunnyCDN struct {
	Zone string
	Key  string
	Base string
}

type WebmentionIO struct {
	Secret string
}

type Webhook struct {
	Secret string
}

type IndieAuth struct {
	Me       string
	Endpoint string
}

func (i *IndieAuth) Authorized() []string {
	return []string{i.Me}
}

func (i *IndieAuth) TokenEndpoint() string {
	return i.Endpoint
}

type Hugo struct {
	Source      string
	Destination string
}

type Telegraph struct {
	Token string
}

type XRay struct {
	Endpoint string
}
