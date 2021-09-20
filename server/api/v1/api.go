package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"log"

	"github.com/go-chi/chi"
)

// ValidBearer is a hardcoded bearer token for demonstration purposes.
var TOKEN string = ""

// HelloResponse is the JSON representation for a customized message
type HelloResponse struct {
	Message string `json:"message"`
}

// GetStatusServerResponse is the JSON representation for an information about user status
type GetStatusServerResponse struct {
	Message 		string 		`json:"message"`
	Status			int64		`json:"status"`
	StatusExplain	string		`json:"status_explain"`
}

type AccessTokenRequest struct {
	GrantType			string		`json:"grant_type"`
	ClientId			string		`json:"client_id"`
	ClientSecret		string		`json:"client_secret"`
}

type AccessTokenResponse struct {
	Message 			string 		`json:"message"`
	AccessToken			string		`json:"access_token"`
}

func jsonResponse(w http.ResponseWriter, data interface{}, c int) {
	dj, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	fmt.Fprintf(w, "%s", dj)
}

// HelloWorld returns a basic "Hello World!" message
func HelloWorld(w http.ResponseWriter, r *http.Request) {
	response := HelloResponse{
		Message: "Hello world!",
	}
	jsonResponse(w, response, http.StatusOK)
}

// HelloName returns a personalized JSON message
func HelloName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	response := HelloResponse{
		Message: fmt.Sprintf("Hello %s!", name),
	}
	jsonResponse(w, response, http.StatusOK)
}

// GetStatus returns an information about user via email
func GetStatus(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		jsonResponse(w, GetStatusServerResponse{ Message: "Empty user email!!!" }, http.StatusBadRequest)
		return
	}

	uri := "https://api.sendpulse.com/oauth/access_token"
	request := AccessTokenRequest{
		GrantType:    "client_credentials",
		ClientId:     "1b99a2a744c16b44dcd697ec23704583",
		ClientSecret: "011b7818ab1b13518887690b0e36affc",
	}
	jsonData, _ := json.Marshal(request)

	response, err := doRequestPOST(uri, jsonData)
	if err != nil {
		jsonResponse(w, GetStatusServerResponse{ Message: err.Error() }, http.StatusBadRequest)
		return
	}

	var accessToken AccessTokenResponse
	err = json.Unmarshal(response, &accessToken)
	if err != nil {
		jsonResponse(w, GetStatusServerResponse{ Message: err.Error() + " Server response: " + string(response) }, http.StatusBadRequest)
		return
	}
	if accessToken.Message != "" {
		jsonResponse(w, GetStatusServerResponse{ Message: accessToken.Message }, http.StatusBadRequest)
		return
	}
	TOKEN = accessToken.AccessToken

	uri = "https://api.sendpulse.com/emails/" + email
	response, err = doRequestGET(uri)
	if err != nil {
		jsonResponse(w, GetStatusServerResponse{ Message: err.Error() }, http.StatusBadRequest)
		return
	}

	var statusMessage GetStatusServerResponse
	_ = json.Unmarshal(response, &statusMessage)
	if statusMessage.Message != "" {
		jsonResponse(w, GetStatusServerResponse{ Message: statusMessage.Message }, http.StatusBadRequest)
		return
	}

	var status []GetStatusServerResponse
	err = json.Unmarshal(response, &status)
	if err != nil {
		jsonResponse(w, GetStatusServerResponse{ Message: err.Error() + " Server response: " + string(response) }, http.StatusBadRequest)
		return
	}

	if len(status) > 0 {
		jsonResponse(w, status[0], http.StatusOK)
	} else {
		jsonResponse(w, GetStatusServerResponse{ Message: "Server returns nil array!!! Server response: " + string(response) }, http.StatusBadRequest)
	}
}

// EnableAuthentication is an example middleware handler that checks for a
// hardcoded bearer token. This can be used to verify session cookies, JWTs
// and more.
func EnableAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RequireAuthentication := []string{"", "{name}"}
		for _, method := range RequireAuthentication {
			if r.URL.Path[8:] == method { //сделать норм проверку для GET с параметрами в url-строке .../method?...
				// Make sure an Authorization header was provided
				token := r.Header.Get("Authorization")
				if token == "" {
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
				token = strings.TrimPrefix(token, "Bearer ")
				// This is where token validation would be done. For this boilerplate,
				// we just check and make sure the token matches a hardcoded string
				if token != TOKEN {
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
			}
		}
		// Assuming that passed, we can execute the authenticated handler
		addCorsHeader(w)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func addCorsHeader(w http.ResponseWriter) {
	headers := w.Header()
	headers.Add("Access-Control-Allow-Origin", "*")
	headers.Add("Vary", "Origin")
	headers.Add("Vary", "Access-Control-Request-Method")
	headers.Add("Vary", "Access-Control-Request-Headers")
	headers.Add("Access-Control-Allow-Headers", "Content-Type, Origin, Accept, token")
	headers.Add("Access-Control-Allow-Methods", "GET, POST,OPTIONS")
}

// NewRouter returns an HTTP handler that implements the routes for the API
func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(EnableAuthentication)

	// Register the API routes
	r.Get("/", HelloWorld)
	r.Get("/{name}", HelloName)
	r.Get("/getStatus", GetStatus)

	return r
}

func httpRequest(request *http.Request) ([]byte, error) {
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func doRequestGET(uri string) ([]byte, error) {
	request, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Authorization", "Bearer " + TOKEN)
	request.Header.Set("Content-Type", "application/json")

	var response []byte
	response, err = httpRequest(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func doRequestPOST(uri string, jsonBytes []byte) ([]byte, error) {
	body := bytes.NewReader(jsonBytes)
	request, err := http.NewRequest("POST", uri, body)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Authorization", "Bearer " + TOKEN)
	request.Header.Set("Content-Type", "application/json")

	var response []byte
	response, err = httpRequest(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}
