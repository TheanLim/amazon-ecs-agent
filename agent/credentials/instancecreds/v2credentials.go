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

package instancecreds

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/amazon-ecs-agent/agent/credentials/providers"
)

// NewV2Credentials returns an AWS SDK v2 CredentialsProvider
func NewV2Credentials(isExternal bool) aws.CredentialsProvider {
	return &v2CredentialsProvider{isExternal: isExternal}
}

type v2CredentialsProvider struct {
	isExternal bool
}

// Retrieve implements the aws.CredentialsProvider interface
func (p *v2CredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	mu.Lock()
	if credentialChain == nil {
		credProviders := defaults.CredProviders(defaults.Config(), defaults.Handlers())
		credProviders = append(credProviders, providers.NewRotatingSharedCredentialsProvider())
		credentialChain = credentials.NewCredentials(&credentials.ChainProvider{
			Providers: credProviders,
		})
	}
	mu.Unlock()

	v1Creds, err := credentialChain.Get()
	if err != nil {
		return aws.Credentials{}, err
	}

	return aws.Credentials{
		AccessKeyID:     v1Creds.AccessKeyID,
		SecretAccessKey: v1Creds.SecretAccessKey,
		SessionToken:    v1Creds.SessionToken,
		Source:         v1Creds.ProviderName,
		CanExpire:      true,
		Expires:        v1Creds.Expires,
	}, nil
}