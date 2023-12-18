package privatedns

import (
	"context"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/cenkalti/backoff/v4"

	azauth "github.com/dapr/components-contrib/common/authentication/azure"
	"github.com/dapr/components-contrib/nameresolution"
	"github.com/dapr/kit/config"
	"github.com/dapr/kit/logger"
)

const (
	defaultZoneName                             = "private.dapr.com"
	zoneName                                    = "zoneName"
	clientID                                    = "clientId"
	tenantID                                    = "tenantId"
	clientSecret                                = "clientSecret"
	subscriptionId                              = "subscriptionId"
	appId                                       = "appId"
	resourceGroup                               = "resourceGroup"
	aciDaprInternalPort                         = 50006
	createOrUpdateRequestContextTimeoutDuration = 20 * time.Second
)

func NewResolver(logger logger.Logger) nameresolution.Resolver {
	return &resolver{
		logger: logger,
		config: resolverConfig{ZoneName: defaultZoneName},
	}
}

type resolver struct {
	logger logger.Logger
	config resolverConfig
}

type resolverConfig struct {
	ZoneName       string
	ClientId       string
	TenantId       string
	ClientSecret   string
	SubscriptionId string
	AppId          string
	ResourceGroup  string
}

// ResolveID implements nameresolution.Resolver.
func (r *resolver) ResolveID(ctx context.Context, req nameresolution.ResolveRequest) (string, error) {
	r.logger.Debug(req.ID)
	return req.ID + "." + r.config.ZoneName + ":" + strconv.Itoa(req.Port), nil
}

// Initialises the resolver with the metadata and creates a DNS Arecord
// Uses either service principal or managed service identity
func (r *resolver) Init(ctx context.Context, metadata nameresolution.Metadata) (err error) {
	r.config, err = getConfig(metadata)
	if err != nil {
		return err
	}

	var credConfig azauth.CredConfigInterface
	if r.config.ClientSecret != "" {
		credConfig = azauth.CredentialsConfig{
			ClientID:     r.config.ClientId,
			ClientSecret: r.config.ClientSecret,
			TenantID:     r.config.TenantId,
			AzureCloud:   &cloud.AzurePublic,
		}
	} else {
		credConfig = azauth.MSIConfig{
			ClientID: r.config.ClientId,
		}
	}

	err = r.registerService(credConfig)

	return err
}

func (r *resolver) registerService(credConfig azauth.CredConfigInterface) error {
	cred, err := credConfig.GetTokenCredential()
	if err != nil {
		return err
	}

	privatednsClientFactory, err := armprivatedns.NewClientFactory(r.config.SubscriptionId, cred, nil)
	if err != nil {
		return err
	}

	recordSetsClient := privatednsClientFactory.NewRecordSetsClient()
	ipAddress, err := getIPAddress()
	if err != nil {
		return err
	}

	createOrUpdateRecordOperation := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), createOrUpdateRequestContextTimeoutDuration)
		defer cancel()
		_, err = recordSetsClient.CreateOrUpdate(ctx, r.config.ResourceGroup, r.config.ZoneName, armprivatedns.RecordTypeA, r.config.AppId, armprivatedns.RecordSet{
			Properties: &armprivatedns.RecordSetProperties{
				ARecords: []*armprivatedns.ARecord{
					{
						IPv4Address: to.Ptr(ipAddress),
					}},
				Metadata: map[string]*string{
					"port": to.Ptr(strconv.Itoa(aciDaprInternalPort)),
				},
				TTL: to.Ptr[int64](3600),
			},
		}, &armprivatedns.RecordSetsClientCreateOrUpdateOptions{IfMatch: nil,
			IfNoneMatch: nil,
		})
		return err
	}

	if r.config.ClientSecret == "" {
		// use backoff to wait for msi sidecar to be set up
		b := getDnsBackoff()
		err = backoff.Retry(createOrUpdateRecordOperation, b)
	} else {
		// no backoff required for service principal
		err = createOrUpdateRecordOperation()
	}

	return err
}

func getConfig(metadata nameresolution.Metadata) (resolverCfg resolverConfig, err error) {
	cfg, err := config.Normalize(metadata.Configuration)
	if err != nil {
		return resolverCfg, err
	}

	config, err := parseConfig(cfg)
	if err != nil {
		return resolverCfg, err
	}
	resolverCfg.ClientId = config.ClientId
	resolverCfg.ClientSecret = config.ClientSecret
	resolverCfg.TenantId = config.TenantId
	resolverCfg.ZoneName = config.ZoneName
	resolverCfg.SubscriptionId = config.SubscriptionId
	resolverCfg.AppId = config.AppId
	resolverCfg.ResourceGroup = config.ResourceGroup

	return resolverCfg, nil
}
