package cors

import(
	"net/http"
)

func EnsureCORSHandler(next http.Handler, cors bool) http.Handler {

		fn := func(rsp http.ResponseWriter, req *http.Request) {

			if cors {
				rsp.Header().Set("Access-Control-Allow-Origin", "*")
			}

			next.ServeHTTP(rsp, req)
		}

		return http.HandlerFunc(fn)
}
