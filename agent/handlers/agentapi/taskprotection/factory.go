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
package taskprotection

import (
	"context"
	"log"
	"os"

	"github.com/aws/amazon-ecs-agent/agent/config"
	"github.com/aws/amazon-ecs-agent/agent/config/ipcompatibility"
	"github.com/aws/amazon-ecs-agent/agent/version"
	ecsapi "github.com/aws/amazon-ecs-agent/ecs-agent/api/ecs"
	ecsclient "github.com/aws/amazon-ecs-agent/ecs-agent/api/ecs/client"
	"github.com/aws/amazon-ecs-agent/ecs-agent/credentials"
	"github.com/aws/amazon-ecs-agent/ecs-agent/httpclient"
	"github.com/aws/amazon-ecs-agent/ecs-agent/logger"
	"github.com/aws/amazon-ecs-agent/ecs-agent/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/smithy-go/logging"
)

// TaskProtectionClientFactory implements TaskProtectionClientFactoryInterface
type TaskProtectionClientFactory struct {
	Region             string
	Endpoint           string
	AcceptInsecureCert bool
	IPCompatibility    ipcompatibility.IPCompatibility
}

// Helper function for retrieving credential from credentials manager and create ecs client
func (factory TaskProtectionClientFactory) NewTaskProtectionClient(
	taskRoleCredential credentials.TaskIAMRoleCredentials,
) (ecsapi.ECSTaskProtectionSDK, error) {
	customLogger := &awsLogger{
		logger: log.New(os.Stdout, "CUSTOM LOGGER - task-protection: ", log.Lshortfile|log.Ltime),
	}

	taskCredential := taskRoleCredential.GetIAMRoleCredentials()

	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithCredentialsProvider(
			awscreds.NewStaticCredentialsProvider(
				taskCredential.AccessKeyID,
				taskCredential.SecretAccessKey,
				taskCredential.SessionToken,
			),
		),
		awsconfig.WithRegion(factory.Region),
		awsconfig.WithHTTPClient(
			httpclient.New(
				ecsclient.RoundtripTimeout,
				factory.AcceptInsecureCert,
				version.String(),
				config.OSType,
			),
		),
		awsconfig.WithLogger(customLogger),
		awsconfig.WithClientLogMode(aws.LogRequest),
	}

	opts = append(opts, factory.getEndpointOptions()...)

	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), opts...)
	if err != nil {
		return nil, err
	}

	ecsClient := ecs.NewFromConfig(cfg)
	return ecsClient, nil
}

func (factory TaskProtectionClientFactory) getEndpointOptions() []func(*awsconfig.LoadOptions) error {
	// When using a custom endpoint, additional endpoint configurations are not supported
	if factory.Endpoint != "" {
		logger.Debug("Configuring TaskProtection custom endpoint")
		return []func(*awsconfig.LoadOptions) error{
			awsconfig.WithBaseEndpoint(utils.AddScheme(factory.Endpoint)),
		}
	}

	// Initialize options slice to support multiple endpoint configurations in future (such as FIPS)
	var opts []func(*awsconfig.LoadOptions) error
	if factory.IPCompatibility.IsIPv6Only() {
		logger.Debug("Configuring TaskProtection DualStack endpoint")
		opts = append(opts, awsconfig.WithUseDualStackEndpoint(aws.DualStackEndpointStateEnabled))
	}
	return opts
}

type awsLogger struct {
	logger *log.Logger
}

func (l awsLogger) Logf(classification logging.Classification, format string, v ...interface{}) {
	l.logger.Printf(format, v...)
}
