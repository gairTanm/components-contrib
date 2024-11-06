package privatedns

import (
	"context"
	"strconv"
	"time"

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
	// auth config
	ClientId string
	// resource config
	ZoneName       string
	SubscriptionId string
	AppId          string
	ResourceGroup  string
	// internalPort bool
}

// ping appId.zoneName:aciDaprInternalPort

// ResolveID implements nameresolution.Resolver.
func (r *resolver) ResolveID(ctx context.Context, req nameresolution.ResolveRequest) (string, error) {
	r.logger.Debug(req.ID)
	return req.ID + "." + r.config.ZoneName + ":" + strconv.Itoa(req.Port), nil
}

// Init Initialises the resolver with the metadata and creates a DNS Arecord
// Uses managed service identity
func (r *resolver) Init(ctx context.Context, metadata nameresolution.Metadata) (err error) {
	r.config, err = getConfig(metadata)
	if err != nil {
		return err
	}

	var credConfig = azauth.MSIConfig{
		ClientID: r.config.ClientId,
	}

	err = r.registerService(credConfig)

	return err
}

func (r *resolver) registerService(credConfig azauth.MSIConfig) error {
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

	// Use backoff to wait for the MSI sidecar to be up and running
	b := getDnsBackoff()
	err = backoff.Retry(createOrUpdateRecordOperation, b)

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
	resolverCfg.ZoneName = config.ZoneName
	resolverCfg.SubscriptionId = config.SubscriptionId
	resolverCfg.AppId = config.AppId
	resolverCfg.ResourceGroup = config.ResourceGroup

	return resolverCfg, nil
}

func (r *resolver) Close() error {
	return nil
}
