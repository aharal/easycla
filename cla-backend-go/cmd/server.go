// Copyright The Linux Foundation and each contributor to CommunityBridge.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"

	"github.com/communitybridge/easycla/cla-backend-go/approval_list"
	"github.com/communitybridge/easycla/cla-backend-go/v2/cla_groups"
	openapi_runtime "github.com/go-openapi/runtime"

	"github.com/communitybridge/easycla/cla-backend-go/projects_cla_groups"

	"github.com/communitybridge/easycla/cla-backend-go/v2/sign"

	"github.com/communitybridge/easycla/cla-backend-go/cla_manager"
	project_service "github.com/communitybridge/easycla/cla-backend-go/v2/project-service"
	user_service "github.com/communitybridge/easycla/cla-backend-go/v2/user-service"

	acs_service "github.com/communitybridge/easycla/cla-backend-go/v2/acs-service"
	organization_service "github.com/communitybridge/easycla/cla-backend-go/v2/organization-service"

	"github.com/communitybridge/easycla/cla-backend-go/github_organizations"
	v2GithubOrganizations "github.com/communitybridge/easycla/cla-backend-go/v2/github_organizations"
	"github.com/communitybridge/easycla/cla-backend-go/v2/metrics"

	"github.com/communitybridge/easycla/cla-backend-go/token"

	"github.com/communitybridge/easycla/cla-backend-go/gerrits"
	v2Gerrits "github.com/communitybridge/easycla/cla-backend-go/v2/gerrits"

	"github.com/aws/aws-sdk-go/service/dynamodb"

	lfxAuth "github.com/LF-Engineering/lfx-kit/auth"
	"github.com/communitybridge/easycla/cla-backend-go/docs"
	"github.com/communitybridge/easycla/cla-backend-go/repositories"
	"github.com/communitybridge/easycla/cla-backend-go/utils"
	v2Docs "github.com/communitybridge/easycla/cla-backend-go/v2/docs"
	v2Events "github.com/communitybridge/easycla/cla-backend-go/v2/events"
	v2Metrics "github.com/communitybridge/easycla/cla-backend-go/v2/metrics"
	v2Repositories "github.com/communitybridge/easycla/cla-backend-go/v2/repositories"
	v2Version "github.com/communitybridge/easycla/cla-backend-go/v2/version"
	"github.com/communitybridge/easycla/cla-backend-go/version"

	"github.com/communitybridge/easycla/cla-backend-go/events"

	"github.com/communitybridge/easycla/cla-backend-go/project"
	v2Project "github.com/communitybridge/easycla/cla-backend-go/v2/project"

	"github.com/communitybridge/easycla/cla-backend-go/users"

	"github.com/communitybridge/easycla/cla-backend-go/signatures"
	v2Signatures "github.com/communitybridge/easycla/cla-backend-go/v2/signatures"

	ini "github.com/communitybridge/easycla/cla-backend-go/init"
	log "github.com/communitybridge/easycla/cla-backend-go/logging"

	"github.com/communitybridge/easycla/cla-backend-go/auth"
	"github.com/communitybridge/easycla/cla-backend-go/company"
	"github.com/communitybridge/easycla/cla-backend-go/config"
	"github.com/communitybridge/easycla/cla-backend-go/docraptor"
	"github.com/communitybridge/easycla/cla-backend-go/gen/models"
	"github.com/communitybridge/easycla/cla-backend-go/gen/restapi"
	"github.com/communitybridge/easycla/cla-backend-go/gen/restapi/operations"
	v2RestAPI "github.com/communitybridge/easycla/cla-backend-go/gen/v2/restapi"
	v2Ops "github.com/communitybridge/easycla/cla-backend-go/gen/v2/restapi/operations"
	"github.com/communitybridge/easycla/cla-backend-go/github"
	"github.com/communitybridge/easycla/cla-backend-go/health"
	"github.com/communitybridge/easycla/cla-backend-go/template"
	"github.com/communitybridge/easycla/cla-backend-go/user"
	v2ClaManager "github.com/communitybridge/easycla/cla-backend-go/v2/cla_manager"
	v2Company "github.com/communitybridge/easycla/cla-backend-go/v2/company"
	v2Health "github.com/communitybridge/easycla/cla-backend-go/v2/health"
	v2Template "github.com/communitybridge/easycla/cla-backend-go/v2/template"

	"github.com/go-openapi/loads"
	"github.com/lytics/logrus"
	"github.com/rs/cors"
	"github.com/savaki/dynastore"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version is the application version - either a git SHA or tag value
	Version string

	// Commit is the application commit hash
	Commit string

	// Branch the build branch
	Branch string

	// BuildDate is the date of the build
	BuildDate string
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the backend server",
	Long:  `Run the backend server which listens for http requests over a given port.`,
	Run:   runServer,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

type combinedRepo struct {
	users.UserRepository
	company.IRepository
	project.ProjectRepository
}

// server function called by environment specific server functions
func server(localMode bool) http.Handler {

	host, err := os.Hostname()
	if err != nil {
		log.Fatalf("unable to get hostname. Error: %v", err)
	}

	var githubOrgValidation = true // default is true/enabled
	githubOrgValidationString := viper.GetString("GH_ORG_VALIDATION")
	if githubOrgValidationString != "" {
		githubOrgValidation, err = strconv.ParseBool(githubOrgValidationString)
		if err != nil {
			log.Fatalf("GH_ORG_VALIDATION value must be a boolean string. Error: %v", err)
		}
	}

	var companyUserValidation = true // default is true/enabled
	companyUserValidationString := viper.GetString("COMPANY_USER_VALIDATION")
	if companyUserValidationString != "" {
		companyUserValidation, err = strconv.ParseBool(companyUserValidationString)
		if err != nil {
			log.Fatalf("COMPANY_USER_VALIDATION value must be a boolean string. Error: %v", err)
		}
	}

	stage := viper.GetString("STAGE")
	dynamodbRegion := ini.GetProperty("DYNAMODB_AWS_REGION")

	log.Infof("Service %s starting...", ini.ServiceName)

	// Show the version and build info
	log.Infof("Name                    : %s", ini.ServiceName)
	log.Infof("Version                 : %s", Version)
	log.Infof("Git commit hash         : %s", Commit)
	log.Infof("Branch                  : %s", Branch)
	log.Infof("Build date              : %s", BuildDate)
	log.Infof("Golang OS               : %s", runtime.GOOS)
	log.Infof("Golang Arch             : %s", runtime.GOARCH)
	log.Infof("DYANAMODB_AWS_REGION    : %s", dynamodbRegion)
	log.Infof("GH_ORG_VALIDATION       : %t", githubOrgValidation)
	log.Infof("COMPANY_USER_VALIDATION : %t", companyUserValidation)
	log.Infof("STAGE                   : %s", stage)
	log.Infof("Service Host            : %s", host)
	log.Infof("Service Port            : %d", *portFlag)

	awsSession, err := ini.GetAWSSession()
	if err != nil {
		log.Panicf("Unable to load AWS session - Error: %v", err)
	}

	configFile, err := config.LoadConfig(configFile, awsSession, stage)
	if err != nil {
		log.Panicf("Unable to load config - Error: %v", err)
	}

	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		logrus.Panicf("Invalid swagger file for initializing EasyCLA v1 - Error: %v", err)
	}

	v2SwaggerSpec, err := loads.Analyzed(v2RestAPI.SwaggerJSON, "")
	if err != nil {
		logrus.Panicf("Invalid swagger file for initializing EasyCLA v2 - Error: %v", err)
	}

	api := operations.NewClaAPI(swaggerSpec)
	v2API := v2Ops.NewEasyclaAPI(v2SwaggerSpec)

	docraptorClient, err := docraptor.NewDocraptorClient(configFile.Docraptor.APIKey, configFile.Docraptor.TestMode)
	if err != nil {
		logrus.Panicf("Unable to setup docraptor client - Error: %v", err)
	}

	authValidator, err := auth.NewAuthValidator(
		configFile.Auth0.Domain,
		configFile.Auth0.ClientID,
		configFile.Auth0.UsernameClaim,
		configFile.Auth0.Algorithm)
	if err != nil {
		logrus.Panic(err)
	}

	token.Init(configFile.Auth0Platform.ClientID, configFile.Auth0Platform.ClientSecret, configFile.Auth0Platform.URL, configFile.Auth0Platform.Audience)
	github.Init(configFile.Github.AppID, configFile.Github.AppPrivateKey, configFile.Github.AccessToken)

	// Our backend repository handlers
	userRepo := user.NewDynamoRepository(awsSession, stage)
	usersRepo := users.NewRepository(awsSession, stage)
	repositoriesRepo := repositories.NewRepository(awsSession, stage)
	gerritRepo := gerrits.NewRepository(awsSession, stage)
	templateRepo := template.NewRepository(awsSession, stage)
	approvalListRepo := approval_list.NewRepository(awsSession, stage)
	companyRepo := company.NewRepository(awsSession, stage)
	signaturesRepo := signatures.NewRepository(awsSession, stage, companyRepo, usersRepo)
	projectClaGroupRepo := projects_cla_groups.NewRepository(awsSession, stage)
	projectRepo := project.NewRepository(awsSession, stage, repositoriesRepo, gerritRepo, projectClaGroupRepo)
	eventsRepo := events.NewRepository(awsSession, stage)
	metricsRepo := metrics.NewRepository(awsSession, stage, configFile.APIGatewayURL, projectClaGroupRepo)
	githubOrganizationsRepo := github_organizations.NewRepository(awsSession, stage)
	claManagerReqRepo := cla_manager.NewRepository(awsSession, stage)

	// Our service layer handlers
	eventsService := events.NewService(eventsRepo, combinedRepo{
		usersRepo,
		companyRepo,
		projectRepo,
	})

	// Initialize the external platform services - these are external APIs that
	// we download the swagger specification, generate the models, and have
	//client helper functions
	user_service.InitClient(configFile.APIGatewayURL, configFile.AcsAPIKey)
	project_service.InitClient(configFile.APIGatewayURL)
	organization_service.InitClient(configFile.APIGatewayURL, eventsService)
	acs_service.InitClient(configFile.APIGatewayURL, configFile.AcsAPIKey)

	usersService := users.NewService(usersRepo, eventsService)
	healthService := health.New(Version, Commit, Branch, BuildDate)
	templateService := template.NewService(stage, templateRepo, docraptorClient, awsSession)
	projectService := project.NewService(projectRepo, repositoriesRepo, gerritRepo, projectClaGroupRepo)
	v2ProjectService := v2Project.NewService(projectService, projectRepo, projectClaGroupRepo)
	companyService := company.NewService(companyRepo, configFile.CorporateConsoleURL, userRepo, usersService)
	v2CompanyService := v2Company.NewService(companyService, signaturesRepo, projectRepo, usersRepo, companyRepo, projectClaGroupRepo, eventsService)
	v2SignService := sign.NewService(configFile.ClaV1ApiURL, companyRepo, projectRepo, projectClaGroupRepo, companyService)
	signaturesService := signatures.NewService(signaturesRepo, companyService, usersService, eventsService, githubOrgValidation)
	v2SignatureService := v2Signatures.NewService(awsSession, configFile.SignatureFilesBucket, projectService, companyService, signaturesService, projectClaGroupRepo)
	v1ClaManagerService := cla_manager.NewService(claManagerReqRepo, companyService, projectService, usersService, signaturesService, eventsService, configFile.CorporateConsoleURL)
	repositoriesService := repositories.NewService(repositoriesRepo, githubOrganizationsRepo, projectClaGroupRepo)
	v2RepositoriesService := v2Repositories.NewService(repositoriesRepo, projectClaGroupRepo, githubOrganizationsRepo)
	v2ClaManagerService := v2ClaManager.NewService(companyService, projectService, v1ClaManagerService, usersService, repositoriesService, v2CompanyService, eventsService, projectClaGroupRepo)
	approvalListService := approval_list.NewService(approvalListRepo, usersRepo, companyRepo, projectRepo, signaturesRepo, configFile.CorporateConsoleURL, http.DefaultClient)
	authorizer := auth.NewAuthorizer(authValidator, userRepo)
	v2MetricsService := metrics.NewService(metricsRepo, projectClaGroupRepo)
	githubOrganizationsService := github_organizations.NewService(githubOrganizationsRepo, repositoriesRepo)
	v2GithubOrganizationsService := v2GithubOrganizations.NewService(githubOrganizationsRepo, repositoriesRepo)
	gerritService := gerrits.NewService(gerritRepo, &gerrits.LFGroup{
		LfBaseURL:    configFile.LFGroup.ClientURL,
		ClientID:     configFile.LFGroup.ClientID,
		ClientSecret: configFile.LFGroup.ClientSecret,
		RefreshToken: configFile.LFGroup.RefreshToken,
	})
	v2ClaGroupService := cla_groups.NewService(projectService, templateService, projectClaGroupRepo, v1ClaManagerService, signaturesService, metricsRepo, gerritService, repositoriesService, eventsService)

	sessionStore, err := dynastore.New(dynastore.Path("/"), dynastore.HTTPOnly(), dynastore.TableName(configFile.SessionStoreTableName), dynastore.DynamoDB(dynamodb.New(awsSession)))
	if err != nil {
		log.Fatalf("Unable to create new Dynastore session - Error: %v", err)
	}
	utils.SetSnsEmailSender(awsSession, configFile.SNSEventTopicARN, configFile.SenderEmailAddress)
	utils.SetS3Storage(awsSession, configFile.SignatureFilesBucket)

	// Setup security handlers
	api.OauthSecurityAuth = authorizer.SecurityAuth
	v2API.LfAuthAuth = lfxAuth.SwaggerAuth

	// Setup our API handlers
	users.Configure(api, usersService, eventsService)
	project.Configure(api, projectService, eventsService, gerritService, repositoriesService, signaturesService)
	v2Project.Configure(v2API, projectService, v2ProjectService, eventsService)
	health.Configure(api, healthService)
	v2Health.Configure(v2API, healthService)
	template.Configure(api, templateService, eventsService)
	v2Template.Configure(v2API, templateService, eventsService)
	github.Configure(api, configFile.Github.ClientID, configFile.Github.ClientSecret, configFile.Github.AccessToken, sessionStore)
	signatures.Configure(api, signaturesService, sessionStore, eventsService)
	v2Signatures.Configure(v2API, projectService, projectRepo, companyService, signaturesService, sessionStore, eventsService, v2SignatureService, projectClaGroupRepo)
	approval_list.Configure(api, approvalListService, sessionStore, signaturesService, eventsService)
	company.Configure(api, companyService, usersService, companyUserValidation, eventsService)
	docs.Configure(api)
	v2Docs.Configure(v2API)
	version.Configure(api, Version, Commit, Branch, BuildDate)
	v2Version.Configure(v2API, Version, Commit, Branch, BuildDate)
	events.Configure(api, eventsService)
	v2Events.Configure(v2API, eventsService, companyRepo, projectClaGroupRepo)
	v2Metrics.Configure(v2API, v2MetricsService, companyRepo)
	github_organizations.Configure(api, githubOrganizationsService, eventsService)
	v2GithubOrganizations.Configure(v2API, v2GithubOrganizationsService, eventsService)
	repositories.Configure(api, repositoriesService, eventsService)
	v2Repositories.Configure(v2API, v2RepositoriesService, eventsService)
	gerrits.Configure(api, gerritService, projectService, eventsService)
	v2Gerrits.Configure(v2API, gerritService, projectService, eventsService, projectClaGroupRepo)
	v2Company.Configure(v2API, v2CompanyService, companyRepo, projectClaGroupRepo, configFile.LFXPortalURL)
	cla_manager.Configure(api, v1ClaManagerService, companyService, projectService, usersService, signaturesService, eventsService, configFile.CorporateConsoleURL)
	v2ClaManager.Configure(v2API, v2ClaManagerService, configFile.LFXPortalURL, projectClaGroupRepo, userRepo)
	sign.Configure(v2API, v2SignService)
	cla_groups.Configure(v2API, v2ClaGroupService, projectService, eventsService)

	userCreaterMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			createUserFromRequest(authorizer, usersService, eventsService, r)
			next.ServeHTTP(w, r)
		})
	}

	// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
	// The middleware executes after routing but before authentication, binding and validation
	middlewareSetupfunc := func(handler http.Handler) http.Handler {
		return setRequestIDHandler(responseLoggingMiddleware(userCreaterMiddleware(handler)))
	}

	v2API.CsvProducer = openapi_runtime.ProducerFunc(func(w io.Writer, data interface{}) error {
		switch v := data.(type) {
		case []byte:
			_, err := w.Write(v)
			return err
		case []string:
			if len(v) == 0 {
				return nil
			}
			_, err := w.Write([]byte(v[0]))
			if err != nil {
				return err
			}
			v = v[1:]
			for _, line := range v {
				_, err = w.Write([]byte("\n"))
				if err != nil {
					return err
				}
				_, err := w.Write([]byte(line))
				if err != nil {
					return err
				}
			}
		default:
			return errors.New("invalid value to CSV producer")
		}
		return nil
	})

	v2API.TextJSONProducer = openapi_runtime.ProducerFunc(func(w io.Writer, data interface{}) error {
		var err error
		switch v := data.(type) {
		case []byte:
			_, err = w.Write(v)
		default:
			b, jerr := json.Marshal(data)
			if jerr != nil {
				return err
			}
			_, err = w.Write(b)
		}
		return err
	})

	// For local mode - we allow anything, otherwise we use the value specified in the config (e.g. AWS SSM)
	var apiHandler http.Handler
	if localMode {
		apiHandler = setupCORSHandlerLocal(
			wrapHandlers(
				// v1 API => /v3, python side is /v1 and /v2
				api.Serve(middlewareSetupfunc), swaggerSpec.BasePath(),
				// v2 API => /v4
				v2API.Serve(middlewareSetupfunc), v2SwaggerSpec.BasePath()))
	} else {
		apiHandler = setupCORSHandler(
			wrapHandlers(
				// v1 API => /v3, python side is /v1 and /v2
				api.Serve(middlewareSetupfunc), swaggerSpec.BasePath(),
				// v2 API => /v4
				v2API.Serve(middlewareSetupfunc), v2SwaggerSpec.BasePath()),
			configFile.AllowedOrigins)
	}
	return apiHandler
}

// setupCORSHandler sets up the CORS logic and creates the middleware HTTP handler
func setupCORSHandler(handler http.Handler, allowedOrigins []string) http.Handler {

	log.Debugf("Allowed origins: %v", allowedOrigins)
	c := cors.New(cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			u, err := url.Parse(origin)
			if err != nil {
				log.Warnf("cors parse origin issue: %v", err)
				return false
			}

			// Ensure the origin is in our allowed list
			allowedOrigin := utils.HostInSlice(u.Hostname(), allowedOrigins)
			if allowedOrigin {
				// localhost with HTTP is allowed
				if strings.HasPrefix(u.Hostname(), "localhost") && u.Scheme == "http" {
					log.Debugf("origin %s with protocol %s is allowed", u.Hostname(), u.Scheme)
					return true
				}

				// non-localhost with HTTPS is allowed
				if !strings.HasPrefix(u.Hostname(), "localhost") && u.Scheme == "https" {
					log.Debugf("origin %s with protocol %s is allowed", u.Hostname(), u.Scheme)
					return true
				}

				log.Debugf("origin %s with protocol %s is NOT allowed", u.Hostname(), u.Scheme)
				return false
			}

			log.Warnf("origin %s is NOT allowed - not in allowed list: %v", u.Hostname(), allowedOrigins)
			return false
		},
		// Enable Debugging for testing, consider disabling in production
		Debug: false,
	})

	return c.Handler(handler)
}

// wrapHandlers routes the request to the appropriate handler
func wrapHandlers(v1 http.Handler, v1BasePath string, v2 http.Handler, v2BasePath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//log.Debugf("Path is: %s", r.URL.Path)
		if strings.HasPrefix(r.URL.Path, v1BasePath) {
			//log.Debugf("Routing to /v3 API handler...")
			v1.ServeHTTP(w, r)
		}
		if strings.HasPrefix(r.URL.Path, v2BasePath) {
			//log.Debugf("Routing to /v2 API handler...")
			v2.ServeHTTP(w, r)
		}
	})
}

// setupCORSHandlerLocal allows all origins and sets up the handler
func setupCORSHandlerLocal(handler http.Handler) http.Handler {

	log.Debug("Allowing all origins")
	c := cors.New(cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		AllowOriginFunc:  func(origin string) bool { return true },
		//AllowOriginFunc:  func(origin string) bool { return true },
		// Enable Debugging for testing, consider disabling in production
		Debug: false,
	})

	return c.Handler(handler)
}

// LoggingResponseWriter is a wrapper around an http.ResponseWriter which captures the
// status code written to the response, so that it can be logged.
type LoggingResponseWriter struct {
	wrapped    http.ResponseWriter
	StatusCode int
	// Response content could also be captured here, but I was only interested in logging the response status code
}

// NewLoggingResponseWriter creates a new logging response writer
func NewLoggingResponseWriter(wrapped http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{wrapped: wrapped}
}

// Header returns the header
func (lrw *LoggingResponseWriter) Header() http.Header {
	return lrw.wrapped.Header()
}

// Write writes the contents
func (lrw *LoggingResponseWriter) Write(content []byte) (int, error) {
	return lrw.wrapped.Write(content)
}

// WriteHeader writes the header
func (lrw *LoggingResponseWriter) WriteHeader(statusCode int) {
	lrw.StatusCode = statusCode
	lrw.wrapped.WriteHeader(statusCode)
}

// setRequestIDHandler adds the x-request-id header, if missing
func setRequestIDHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the x-request-id header value if it doesn't exist...
		if r.Header.Get("x-request-id") == "" {
			requestID, err := uuid.NewV4()
			if err != nil {
				log.Warnf("unable to generate a UUID for x-request-id header, error: %v", err)
			} else {
				r.Header.Set("x-request-id", requestID.String())
			}
		}
		next.ServeHTTP(w, r)
	})
}

// responseLoggingMiddleware logs the responses from API endpoints
func responseLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(NewLoggingResponseWriter(w), r)
		if r.Response != nil {
			log.Debugf("%s %s, response code: %d response status: %s",
				r.Method, r.URL.String(), r.Response.StatusCode, r.Response.Status)
		} else {
			log.Debugf("%s %s", r.Method, r.URL.String())
		}
	})
}

// create user form http authorization token
// this function creates user if user does not exist and token is valid
func createUserFromRequest(authorizer auth.Authorizer, usersService users.Service, eventsService events.Service, r *http.Request) {
	btoken := r.Header.Get("Authorization")
	if btoken == "" {
		return
	}
	t := strings.Split(btoken, " ")
	if len(t) != 2 {
		return
	}
	token := t[1]
	// parse user from authtoken
	claUser, err := authorizer.SecurityAuth(token, []string{})
	if err != nil {
		log.Error("createUserFromRequest: parsing failed", err)
		return
	}
	// search if user exist in database
	userModel, err := usersService.GetUserByLFUserName(claUser.LFUsername)
	if err != nil {
		log.Error("createUserFromRequest: searching user by lf-username failed", err)
		return
	}
	if userModel != nil {
		return
	}
	newUser := &models.User{
		LfEmail:    claUser.LFEmail,
		LfUsername: claUser.LFUsername,
		Username:   claUser.Name,
	}
	log.WithField("user", newUser).Debug("creating new user")
	userModel, err = usersService.CreateUser(newUser, nil)
	if err != nil {
		log.WithField("user", newUser).Error("creating new user failed")
		return
	}
	eventsService.LogEvent(&events.LogEventArgs{
		EventType: events.UserCreated,
		UserID:    userModel.UserID,
		UserModel: userModel,
		EventData: &events.UserCreatedEventData{},
	})
}
