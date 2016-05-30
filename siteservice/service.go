package siteservice

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"

	"github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/itsyouonline/identityserver/communication"
	"github.com/itsyouonline/identityserver/siteservice/apiconsole"
	"github.com/itsyouonline/identityserver/siteservice/website/packaged/assets"
	"github.com/itsyouonline/identityserver/siteservice/website/packaged/components"
	"github.com/itsyouonline/identityserver/siteservice/website/packaged/html"
	"github.com/itsyouonline/identityserver/siteservice/website/packaged/thirdpartyassets"
	"github.com/itsyouonline/identityserver/specifications"
	"github.com/itsyouonline/identityserver/validation"

	log "github.com/Sirupsen/logrus"
)

//Service is the identityserver http service
type Service struct {
	Sessions                     map[SessionType]*sessions.CookieStore
	smsService                   *communication.SMSService
	phonenumberValidationService *validation.IYOPhonenumberValidationService
}

//NewService creates and initializes a Service
func NewService(cookieSecret string, smsService *communication.SMSService) (service *Service) {
	service = &Service{smsService: smsService}
	p := &validation.IYOPhonenumberValidationService{SMSService: smsService}
	service.phonenumberValidationService = p
	service.initializeSessions(cookieSecret)
	return
}

//InitModels initialize persistance models
func (service *Service) InitModels() {
	service.initLoginModels()
	service.initRegistrationModels()
}

//AddRoutes registers the http routes with the router
func (service *Service) AddRoutes(router *mux.Router) {
	router.Methods("GET").Path("/").HandlerFunc(service.HomePage)
	//Registration form
	router.Methods("GET").Path("/register").HandlerFunc(service.ShowRegistrationForm)
	router.Methods("POST").Path("/register").HandlerFunc(service.ProcessRegistrationForm)
	router.Methods("GET").Path("/registersmsconfirmation").HandlerFunc(service.ShowPhonenumberConfirmationForm)
	router.Methods("POST").Path("/registersmsconfirmation").HandlerFunc(service.ProcessPhonenumberConfirmationForm)
	router.Methods("GET").Path("/registerresendsms").HandlerFunc(service.ShowResendPhonenumberConfirmation)
	router.Methods("POST").Path("/registerresendsms").HandlerFunc(service.ResendPhonenumberConfirmation)
	router.Methods("GET").Path("/phonevalidation").HandlerFunc(service.PhonenumberValidation)
	router.Methods("GET").Path("/registrationsmsconfirmed").HandlerFunc(service.CheckRegistrationSMSConfirmation)
	//Login forms
	router.Methods("GET").Path("/login").HandlerFunc(service.ShowLoginForm)
	router.Methods("POST").Path("/login").HandlerFunc(service.ProcessLoginForm)
	router.Methods("GET").Path("/logintotpconfirmation").HandlerFunc(service.ShowTOTPConfirmationForm)
	router.Methods("POST").Path("/logintotpconfirmation").HandlerFunc(service.ProcessTOTPConfirmation)
	router.Methods("GET").Path("/loginsmsconfirmation").HandlerFunc(service.Show2FASMSConfirmationForm)
	router.Methods("POST").Path("/loginsmsconfirmation").HandlerFunc(service.Process2FASMSConfirmation)
	router.Methods("GET").Path("/sc").HandlerFunc(service.MobileSMSConfirmation)
	router.Methods("GET").Path("/loginsmsconfirmed").HandlerFunc(service.Check2FASMSConfirmation)
	//Authorize form
	router.Methods("GET").Path("/authorize").HandlerFunc(service.ShowAuthorizeForm)
	//Facebook callback
	router.Methods("GET").Path("/facebook_callback").HandlerFunc(service.FacebookCallback)
	//Github callback
	router.Methods("GET").Path("/github_callback").HandlerFunc(service.GithubCallback)
	//Logout link
	router.Methods("GET").Path("/logout").HandlerFunc(service.Logout)
	//Error page
	router.Methods("GET").Path("/error").HandlerFunc(service.ErrorPage)
	router.Methods("GET").Path("/error{errornumber}").HandlerFunc(service.ErrorPage)

	//host the assets used in the htmlpages
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(
		&assetfs.AssetFS{Asset: assets.Asset, AssetDir: assets.AssetDir, AssetInfo: assets.AssetInfo})))
	router.PathPrefix("/thirdpartyassets/").Handler(http.StripPrefix("/thirdpartyassets/", http.FileServer(
		&assetfs.AssetFS{Asset: thirdpartyassets.Asset, AssetDir: thirdpartyassets.AssetDir, AssetInfo: thirdpartyassets.AssetInfo})))
	router.PathPrefix("/components/").Handler(http.StripPrefix("/components/", http.FileServer(
		&assetfs.AssetFS{Asset: components.Asset, AssetDir: components.AssetDir, AssetInfo: components.AssetInfo})))

	//host the apidocumentation
	router.Methods("GET").Path("/apidocumentation").HandlerFunc(service.APIDocs)
	router.PathPrefix("/apidocumentation/raml/").Handler(http.StripPrefix("/apidocumentation/raml", http.FileServer(
		&assetfs.AssetFS{Asset: specifications.Asset, AssetDir: specifications.AssetDir, AssetInfo: specifications.AssetInfo})))
	router.PathPrefix("/apidocumentation/").Handler(http.StripPrefix("/apidocumentation/", http.FileServer(
		&assetfs.AssetFS{Asset: apiconsole.Asset, AssetDir: apiconsole.AssetDir, AssetInfo: apiconsole.AssetInfo})))

}

const (
	mainpageFileName          = "index.html"
	homepageFileName          = "home.html"
	errorpageFilename         = "error.html"
	apidocsPageFilename       = "apidocumentation.html"
	smsMobileConfirmationPage = "smsconfirmation.html"
)

//ShowPublicSite shows the public website
func (service *Service) ShowPublicSite(w http.ResponseWriter, request *http.Request) {
	htmlData, err := html.Asset(mainpageFileName)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Write(htmlData)
}

//APIDocs shows the api documentation
func (service *Service) APIDocs(w http.ResponseWriter, request *http.Request) {
	htmlData, err := html.Asset(apidocsPageFilename)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Write(htmlData)
}

//HomePage shows the home page when logged in, if not, delegate to showing the public website
func (service *Service) HomePage(w http.ResponseWriter, request *http.Request) {

	loggedinuser, err := service.GetLoggedInUser(request)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if loggedinuser == "" {
		service.ShowPublicSite(w, request)
		return
	}

	htmlData, err := html.Asset(homepageFileName)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	sessions.Save(request, w)
	w.Write(htmlData)
}

//Logout logs out the user and redirect to the homepage
//TODO: csrf protection, really important here!
func (service *Service) Logout(w http.ResponseWriter, request *http.Request) {
	service.SetLoggedInUser(w, request, "")
	sessions.Save(request, w)
	http.Redirect(w, request, "", http.StatusFound)
}

//ErrorPage shows the errorpage
func (service *Service) ErrorPage(w http.ResponseWriter, request *http.Request) {
	errornumber := mux.Vars(request)["errornumber"]
	log.Debug("Errorpage requested for error ", errornumber)

	htmlData, err := html.Asset(errorpageFilename)
	if err != nil {
		log.Error("ERROR rendering error page: ", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	htmlData = bytes.Replace(htmlData, []byte(`500`), []byte(errornumber), 1)
	w.Write(htmlData)
}

//renderSMSConfirmationPage renders a small mobile friendly confirmation page after a user follows a link in an sms
func (service *Service) renderSMSConfirmationPage(w http.ResponseWriter, request *http.Request, text string) {
	htmlData, err := html.Asset(smsMobileConfirmationPage)
	if err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	htmlData = bytes.Replace(htmlData, []byte(`{{ text }}`), []byte(text), 1)
	sessions.Save(request, w)
	w.Write(htmlData)
}

func redirectToDifferentPage(w http.ResponseWriter, request *http.Request, keepQueryParams bool, from string, to string) {
	log.Debugf("Redirecting from %s to %s", from, to)
	sessions.Save(request, w)
	redirectTo := to
	if keepQueryParams {
		u, _ := url.Parse(request.RequestURI)
		path := strings.TrimSuffix(u.Path, from)
		path += to
		u.Path = path
		redirectTo = u.RequestURI()
	}
	http.Redirect(w, request, redirectTo, http.StatusFound)
}
