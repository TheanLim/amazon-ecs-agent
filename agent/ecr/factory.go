// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Package ecr helps generate clients to talk to the ECR API
package ecr

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"../../ecs-agent/credentials/instancecreds"
	apicontainer "github.com/aws/amazon-ecs-agent/agent/api/container"
	"github.com/aws/amazon-ecs-agent/agent/config"
	agentversion "github.com/aws/amazon-ecs-agent/agent/version"
	"github.com/aws/amazon-ecs-agent/ecs-agent/credentials"
	"github.com/aws/amazon-ecs-agent/ecs-agent/httpclient"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

// ECRFactory defines the interface to produce an ECR SDK client
type ECRFactory interface {
	GetClient(*apicontainer.ECRAuthData) (ECRClient, error)
}

type ecrFactory struct {
	httpClient *http.Client
}

const (
	roundtripTimeout = 5 * time.Second
)

// NewECRFactory returns an ECRFactory capable of producing ECRSDK clients
func NewECRFactory(acceptInsecureCert bool) ECRFactory {
	return &ecrFactory{
		httpClient: httpclient.New(roundtripTimeout, acceptInsecureCert, agentversion.String(), config.OSType),
	}
}

// GetClient creates the ECR SDK client based on the authdata
func (factory *ecrFactory) GetClient(authData *apicontainer.ECRAuthData) (ECRClient, error) {
	clientConfig, err := getClientConfig(factory.httpClient, authData)
	if err != nil {
		return &ecrClient{}, err
	}

	return factory.newClient(clientConfig), nil
}

// getClientConfig returns the config for the ecr client based on authData
func getClientConfig(httpClient *http.Client, authData *apicontainer.ECRAuthData) (*aws.Config, error) {
	ctx := context.Background()
	cfg, err := awsconfig.LoadDefaultConfig(ctx, 
		awsconfig.WithRegion(authData.Region),
		awsconfig.WithHTTPClient(httpClient),
		awsconfig.WithUseDualStackEndpoint(aws.DualStackEndpointStateEnabled),
	)
	if err != nil {
		return nil, err
	}

	if authData.EndpointOverride != "" {
		cfg.BaseEndpoint = aws.String(authData.EndpointOverride)
	}

	if authData.UseExecutionRole {
		if authData.GetPullCredentials() == (credentials.IAMRoleCredentials{}) {
			return nil, fmt.Errorf("container uses execution credentials, but the credentials are empty")
		}
		staticCreds := awscreds.NewStaticCredentialsProvider(
			authData.GetPullCredentials().AccessKeyID,
			authData.GetPullCredentials().SecretAccessKey,
			authData.GetPullCredentials().SessionToken,
		)
		cfg.Credentials = staticCreds
	} else {
		cfg.Credentials = instancecreds.NewV2Credentials(false)
	}
	
	return &cfg, nil
}

func (factory *ecrFactory) newClient(cfg *aws.Config) ECRClient {
	ecrClient := ecr.NewFromConfig(*cfg)
	return NewECRClient(ecrClient)
}
