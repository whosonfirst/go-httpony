package inject

/* TOO SOON */

import (
	"github.com/whosonfirst/go-httpony/rewrite"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

func NewInjectRewriter(scripts []string) (*InjectRewriter, error) {
	t := InjectRewriter{scripts}
	return &t, nil
}

type InjectRewriter struct {
	scripts []string
}

func (t *InjectRewriter) SetKey(key string, value interface{}) error {
	return nil
}

func (t *InjectRewriter) Rewrite(node *html.Node, writer io.Writer) error {

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
	}

	f(node, writer)

	html.Render(writer, node)
	return nil
}

func NewInjectProvider(writer *InjectRewriter, docroot string) (*InjectProvider, error) {

	i := InjectProvider{
		Writer:  writer,
		docroot: docroot,
	}

	return &i, nil
}

type InjectProvider struct {
	Writer  *InjectRewriter
	docroot string
}

func (s *InjectProvider) InjectHandler(next http.Handler) http.Handler {

	re_html, _ := regexp.Compile(`/(?:(?:.*).html)?$`)

	rewriter, _ := rewrite.NewHTMLRewriterHandler(s.Writer)

	fn := func(rsp http.ResponseWriter, req *http.Request) {

		url := req.URL
		path := url.Path

		/*

			Because there doesn't appear to be anyway to pass a handler func to http.FileServer
			to intercept the _response_ data so we have to mirror the functionality of the file
			server itself here... (20160630/thisisaaronland)

		*/

		if re_html.MatchString(path) {

			log.Printf("INJECT %s\n", path)

			abs_path := filepath.Join(s.docroot, path)

			info, err := os.Stat(abs_path)

			if err != nil {
				next.ServeHTTP(rsp, req)
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

		}

		next.ServeHTTP(rsp, req)
	}

	return http.HandlerFunc(fn)
}
