package sso

import (
	"net/url"

	"github.com/determined-ai/determined/master/internal/license"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/plugin/oauth"
	"github.com/determined-ai/determined/master/internal/plugin/oidc"
	"github.com/determined-ai/determined/master/internal/plugin/saml"
	"github.com/determined-ai/determined/master/internal/plugin/scim"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// AddProviderInfoToMasterResponse modifies passed in master response adds sso
// provider information. In OSS this is a no-op. While having two functions that
// just set a field is somewhat awkward it avoids having to have OSS have any
// requirements that masterResp/masterInfo has a field defined for provider info.
func AddProviderInfoToMasterResponse(config *config.Config, masterResp *apiv1.GetMasterResponse) {
	for _, p := range getProviders(config) {
		masterResp.SsoProviders = append(masterResp.SsoProviders,
			&apiv1.SSOProvider{Name: p.Name, SsoUrl: p.SSOInitiateURL, Type: p.Type, AlwaysRedirect: p.AlwaysRedirect})
	}
}

// AddProviderInfoToMasterInfo modifies passed in master info adds sso
// provider information. In OSS this is a no-op.
func AddProviderInfoToMasterInfo(config *config.Config, masterInfo *aproto.MasterInfo) {
	masterInfo.SSOProviders = getProviders(config)
}

func getProviders(config *config.Config) []aproto.SSOProviderInfo {
	var ssoProviderInfo []aproto.SSOProviderInfo
	if config.SAML.Enabled {
		// Parsing of the URL is checked during validation, so we can drop this error.
		u, _ := url.Parse(config.SAML.IDPRecipientURL)
		u.Path = saml.SAMLRoot + saml.InitiatePath
		ssoProviderInfo = append(ssoProviderInfo, aproto.SSOProviderInfo{
			SSOInitiateURL: u.String(),
			Name:           config.SAML.Provider,
			Type:           "SAML",
			AlwaysRedirect: config.SAML.AlwaysRedirect,
		})
	}

	if config.OIDC.Enabled {
		u, _ := url.Parse(config.OIDC.IDPRecipientURL)
		u.Path = oidc.OidcRoot + oidc.InitiatePath
		name := config.OIDC.Provider
		if len(name) == 0 {
			name = "SSO"
		}

		ssoProviderInfo = append(ssoProviderInfo, aproto.SSOProviderInfo{
			SSOInitiateURL: u.String(),
			Name:           name,
			Type:           "OIDC",
			AlwaysRedirect: config.OIDC.AlwaysRedirect,
		})
	}

	if config.DetCloud.Enabled {
		ssoProviderInfo = append(ssoProviderInfo, aproto.SSOProviderInfo{
			SSOInitiateURL: config.DetCloud.LoginURL,
			Name:           "det-cloud",
			Type:           "DetCloud",
			AlwaysRedirect: config.DetCloud.AlwaysRedirect,
		})
	}
	return ssoProviderInfo
}

func getMasterURL(config *config.Config) (*url.URL, error) {
	// DET-2035: move master URL field out of provisioner and avoid brittle
	// inference of the master URL.
	s := "http://localhost:8080"
	for _, pool := range config.ResourceManagers()[0].ResourcePools {
		if pool.Provider != nil {
			s = pool.Provider.MasterURL
			break
		}
	}

	u, err := url.Parse(s)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return u, nil
}

// RegisterAPIHandlers registers needed API handlers
// determined by master config. In OSS this is just a no-op.
func RegisterAPIHandlers(config *config.Config, db *db.PgDB, echo *echo.Echo) error {
	masterURL, err := getMasterURL(config)
	if err != nil {
		return errors.Wrap(err, "couldn't parse masterURL")
	}

	var oauthService *oauth.Service
	if config.Scim.Auth.OAuthConfig != nil {
		license.RequireLicense("Oauth")
		oauthService, err = oauth.New(user.GetService(), db)
		if err != nil {
			return err
		}
		oauth.RegisterAPIHandler(echo, oauthService)
		log.Infof("OAuth is enabled at %s%s", masterURL, oauth.Root)
	} else {
		log.Info("OAuth is disabled")
	}

	if config.Scim.Enabled {
		license.RequireLicense("SCIM")
		scim.RegisterAPIHandler(echo, db, &config.Scim, masterURL, oauthService)
		log.Infof("SCIM is enabled at %v/scim/v2", masterURL)
	} else {
		log.Info("SCIM is disabled")
	}

	if config.SAML.Enabled {
		license.RequireLicense("SAML")
		samlService, err := saml.New(db, config.SAML)
		if err != nil {
			return errors.Wrap(err, "error creating SAML service")
		}
		saml.RegisterAPIHandler(echo, samlService)
		log.Info("SAML is enabled")
	} else {
		log.Info("SAML is disabled")
	}

	if config.OIDC.Enabled {
		license.RequireLicense("OIDC")
		var pachEnabled bool
		if config.Integrations.Pachyderm.Address != "" {
			pachEnabled = true
		} else {
			pachEnabled = false
		}
		oidcService, err := oidc.New(db, config.OIDC, pachEnabled)
		if err != nil {
			return errors.Wrap(err, "error creating OIDC service")
		}
		oidc.RegisterAPIHandler(echo, oidcService)
		log.Info("OIDC is enabled")
	} else {
		log.Info("OIDC is disabled")
	}

	if config.DetCloud.Enabled {
		license.RequireLicense("DetCloud")
		log.Info("Det Cloud is enabled")
	} else {
		log.Info("Det Cloud is disabled")
	}
	return nil
}
