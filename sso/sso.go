package sso

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/whosonfirst/go-httpony/crypto"
	"github.com/whosonfirst/go-httpony/rewrite"
	"github.com/vaughan0/go-ini"
	"golang.org/x/net/html"
	"golang.org/x/oauth2"	
)

func NewSSORewriter() (*SSORewriter, error) {
	t := SSORewriter{}
	return &t, nil
}

type SSORewriter struct {
	rewrite.HTMLRewriter
	Request *http.Request
	Secret  string
}

func (t *SSORewriter) SetKey(key string, value interface{}) error {

	if key == "request" {
		req := value.(*http.Request)
		t.Request = req
	}

	if key == "secret" {
		t.Secret = value.(string)
	}

	return nil
}

func (t *SSORewriter) Rewrite(node *html.Node, writer io.Writer) error {

	var f func(node *html.Node, writer io.Writer)

	f = func(n *html.Node, w io.Writer) {

		if n.Type == html.ElementNode && n.Data == "body" {

			t_cookie, _ := t.Request.Cookie("t")

			crypt, _ := crypto.NewCrypt(t.Secret)
			token, _ := crypt.Decrypt(t_cookie.Value)

			token_ns := ""
			token_key := "data-api-access-token"
			token_value := token

			token_attr := html.Attribute{token_ns, token_key, token_value}
			n.Attr = append(n.Attr, token_attr)

			endpoint_ns := ""
			endpoint_key := "data-api-endpoint"
			endpoint_value := "fix-me"

			endpoint_attr := html.Attribute{endpoint_ns, endpoint_key, endpoint_value}
			n.Attr = append(n.Attr, endpoint_attr)
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c, w)
		}
	}

	f(node, writer)

	html.Render(writer, node)
	return nil
}

type SSOHandler struct {

}

func NewSSOHandler(sso_config string) (*SSOHandler, error){

	sso_cfg, err = ini.LoadFile(*sso_config)

	if err != nil {
		return err
	}

	oauth_client, _ := sso_cfg.Get("oauth", "client_id")
	oauth_secret, _ := sso_cfg.Get("oauth", "client_secret")
	oauth_auth_url, _ := sso_cfg.Get("oauth", "auth_url")
	oauth_token_url, _ := sso_cfg.Get("oauth", "token_url")

	// shrink to 32 characters

	hash := md5.New()
	hash.Write([]byte(oauth_secret))
	crypto_secret := hex.EncodeToString(hash.Sum(nil))

	crypto, err := crypto.NewCrypt(crypto_secret)

	if err != nil {
		return err
	}

	// writer.SetKey("secret", crypto_secret)

	redirect_url := "fix me"

	conf := &oauth2.Config{
		ClientID:     oauth_client,
		ClientSecret: oauth_secret,
		Scopes:       []string{},
		Endpoint: oauth2.Endpoint{
			AuthURL:  oauth_auth_url,
			TokenURL: oauth_token_url,
		},
		RedirectURL: redirect_url,
	}

}

func (s *SSOHandler) Provider() http.HandleFunc {

}
