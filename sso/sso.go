package sso

/*

This is still wet-paint and a bit of hot mess in places. It works but it's not pretty - specifically
the handling of variables needed by both SSORewriter and SSOProvider. I am not sure what the correct
approach is right now beyond just holding my nose and being happy it works at all... (20160701/thisisaaronland)

*/

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/vaughan0/go-ini"
	"github.com/whosonfirst/go-httpony/crypto"
	"github.com/whosonfirst/go-httpony/rewrite"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/oauth2"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func NewSSORewriter(crypt *crypto.Crypt) (*SSORewriter, error) {
	t := SSORewriter{Crypto: crypt}
	return &t, nil
}

type SSORewriter struct {
	rewrite.HTMLRewriter
	Request     *http.Request
	Crypto      *crypto.Crypt
	cookie_name string
	scripts     []string
}

func (t *SSORewriter) SetKey(key string, value interface{}) error {

	if key == "request" {
		req := value.(*http.Request)
		t.Request = req
	}

	if key == "cookie_name" {
		cookie_name := value.(string)
		t.cookie_name = cookie_name
	}

	if key == "scripts" {
		scripts := value.([]string)
		t.scripts = scripts
	}

	return nil
}

func (t *SSORewriter) Rewrite(node *html.Node, writer io.Writer) error {

	var f func(node *html.Node, writer io.Writer)

	f = func(n *html.Node, w io.Writer) {

		if n.Type == html.ElementNode && n.Data == "head" {

			if len(t.scripts) > 0 {

				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c, w)
				}

				for _, src := range t.scripts {
					script_type := html.Attribute{"", "type", "text/javascript"}
					script_src := html.Attribute{"", "src", src}

					script := html.Node{
						Type:      html.ElementNode,
						DataAtom:  atom.Script,
						Data:      "script",
						Namespace: "",
						Attr:      []html.Attribute{script_type, script_src},
					}

					n.AppendChild(&script)
				}
			}
		}

		if n.Type == html.ElementNode && n.Data == "body" {

			api_endpoint := ""
			api_token := ""
			api_ok := false

			auth_cookie, err := t.Request.Cookie(t.cookie_name)

			if err == nil {

				cookie, err := t.Crypto.Decrypt(auth_cookie.Value)

				if err != nil {
					log.Printf("failed to decrypt cookie because %v\n", err)
				} else {

					stuff := strings.Split(cookie, "#")

					if len(stuff) != 2 {
						log.Printf("failed to parse cookie - expected (2) parts and got %d\n", len(stuff))
					} else {

						api_endpoint = stuff[0]
						api_token = stuff[1]
						api_ok = true
					}
				}
			}

			if api_ok {

				token_ns := ""
				token_key := "data-api-access-token"
				token_value := api_token

				token_attr := html.Attribute{token_ns, token_key, token_value}
				n.Attr = append(n.Attr, token_attr)

				endpoint_ns := ""
				endpoint_key := "data-api-endpoint"
				endpoint_value := api_endpoint

				endpoint_attr := html.Attribute{endpoint_ns, endpoint_key, endpoint_value}
				n.Attr = append(n.Attr, endpoint_attr)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c, w)
		}
	}

	f(node, writer)

	html.Render(writer, node)
	return nil
}

type SSOProvider struct {
	Crypto       *crypto.Crypt
	Writer       *SSORewriter
	OAuth        *oauth2.Config
	endpoint     string
	api_endpoint string
	docroot      string
	cookie_name  string
	tls_enable   bool
}

func NewSSOProvider(sso_config string, endpoint string, docroot string, tls_enable bool) (*SSOProvider, error) {

	sso_cfg, err := ini.LoadFile(sso_config)

	if err != nil {
		return nil, err
	}

	required_oauth := []string{"client_id", "client_secret", "auth_url", "token_url", "api_url", "scopes"}
	required_www := []string{"cookie_name", "cookie_secret"}

	required := make(map[string][]string)

	required["oauth"] = required_oauth
	required["www"] = required_www

	for src, keys := range required {

		for _, key := range keys {

			value, ok := sso_cfg.Get(src, key)

			if !ok {
				return nil, errors.New(fmt.Sprintf("Missing %s key %s", src, key))
			}

			if value == "" {
				return nil, errors.New(fmt.Sprintf("Invalid %s key %s", src, key))
			}
		}
	}

	oauth_client, _ := sso_cfg.Get("oauth", "client_id")
	oauth_secret, _ := sso_cfg.Get("oauth", "client_secret")
	oauth_auth_url, _ := sso_cfg.Get("oauth", "auth_url")
	oauth_token_url, _ := sso_cfg.Get("oauth", "token_url")
	oauth_api_url, _ := sso_cfg.Get("oauth", "api_url")

	oauth_scopes_str, _ := sso_cfg.Get("oauth", "scopes")
	oauth_scopes := strings.Split(oauth_scopes_str, ",")

	if len(oauth_scopes) == 0 {
		return nil, errors.New("Missing scopes")
	}

	cookie_name, _ := sso_cfg.Get("www", "cookie_name")
	cookie_secret, _ := sso_cfg.Get("www", "cookie_secret")

	scripts := make([]string, 0)

	scripts_str, ok := sso_cfg.Get("www", "scripts")

	if ok {
		scripts = strings.Split(scripts_str, ",")
	}

	// shrink to 32 characters

	hash := md5.New()
	hash.Write([]byte(cookie_secret))
	crypto_secret := hex.EncodeToString(hash.Sum(nil))

	crypt, err := crypto.NewCrypt(crypto_secret)

	if err != nil {
		return nil, err
	}

	writer, err := NewSSORewriter(crypt)

	if err != nil {
		return nil, err
	}

	writer.SetKey("cookie_name", cookie_name)
	writer.SetKey("scripts", scripts)

	redirect_url := fmt.Sprintf("http://%s/auth/", endpoint)

	if tls_enable {
		redirect_url = fmt.Sprintf("https://%s/auth/", endpoint)
	}

	conf := &oauth2.Config{
		ClientID:     oauth_client,
		ClientSecret: oauth_secret,
		Scopes:       oauth_scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  oauth_auth_url,
			TokenURL: oauth_token_url,
		},
		RedirectURL: redirect_url,
	}

	pr := SSOProvider{
		Crypto:       crypt,
		Writer:       writer,
		OAuth:        conf,
		endpoint:     endpoint,
		api_endpoint: oauth_api_url,
		docroot:      docroot,
		cookie_name:  cookie_name,
		tls_enable:   tls_enable,
	}

	return &pr, nil
}

func (s *SSOProvider) SSOHandler(next http.Handler) http.Handler {

	re_signin, _ := regexp.Compile(`/signin/?$`)
	re_auth, _ := regexp.Compile(`/auth/?$`)
	re_html, _ := regexp.Compile(`/(?:(?:.*).html)?$`)

	rewriter, _ := rewrite.NewHTMLRewriterHandler(s.Writer)

	fn := func(rsp http.ResponseWriter, req *http.Request) {

		url := req.URL
		path := url.Path

		state := ""

		if re_signin.MatchString(path) {

			_, err := req.Cookie("t")

			if err == nil {
				http.Redirect(rsp, req, "/", 302) // FIXME - do not simply redirect to /
				return
			}

			url := s.OAuth.AuthCodeURL(state, oauth2.AccessTypeOnline)
			http.Redirect(rsp, req, url, 302)
			return
		}

		if re_auth.MatchString(path) {

			query := req.URL.Query()
			code := query.Get("code")

			if code == "" {
				http.Error(rsp, "Missing code parameter", http.StatusBadRequest)
				return
			}

			/*

				for example:

				{
					"access_token": "TOKEN",
					"scope": "write",
					"expires": 1467477951,
					"expires_in": 79931
				}

			*/

			token, err := s.OAuth.Exchange(oauth2.NoContext, code)

			if err != nil {
				http.Error(rsp, err.Error(), http.StatusBadRequest)
				return
			}

			stuff := []string{s.api_endpoint, token.AccessToken}
			cookie := strings.Join(stuff, "#")

			t, err := s.Crypto.Encrypt(cookie)

			if err != nil {
				http.Error(rsp, err.Error(), http.StatusInternalServerError)
				return
			}

			auth_cookie := http.Cookie{Name: s.cookie_name, Value: t, Expires: token.Expiry, Path: "/", HttpOnly: true, Secure: s.tls_enable}
			http.SetCookie(rsp, &auth_cookie)

			http.Redirect(rsp, req, "/", 302) // FIXME - do not simply redirect to /
			return
		}

		/*

			Because there doesn't appear to be anyway to pass a handler func to http.FileServer
			to intercept the _response_ data so we have to mirror the functionality of the file
			server itself here... (20160630/thisisaaronland)

		*/

		if re_html.MatchString(path) {

			abs_path := filepath.Join(s.docroot, path)

			info, err := os.Stat(abs_path)

			if err != nil {
				http.Error(rsp, err.Error(), http.StatusInternalServerError)
				return
			}

			if info.IsDir() {
				abs_path = filepath.Join(abs_path, "index.html")
			}

			reader, err := os.Open(abs_path)

			if err != nil {
				http.Error(rsp, err.Error(), http.StatusInternalServerError)
				return
			}

			handler := rewriter.Handler(reader)

			handler.ServeHTTP(rsp, req)
			return
		}

		next.ServeHTTP(rsp, req)
	}

	return http.HandlerFunc(fn)
}
