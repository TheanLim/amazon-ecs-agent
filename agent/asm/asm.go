// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//    http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package asm

import (
	"encoding/json"
	"fmt"

	"github.com/aws/amazon-ecs-agent/ecs-agent/logger"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/cihub/seelog"
	"github.com/docker/docker/api/types/registry"
	"github.com/pkg/errors"
)

// AuthDataValue is the schema for
// the SecretStringValue returned by ASM
type AuthDataValue struct {
	Username *string
	Password *string
}

func resourceInitializationErrMsg(secretID string) string {
	return fmt.Sprintf(
		`ResourceNotFoundException: The task can't retrieve the secret with ARN %sfrom AWS Secrets Manager. Check whether the secret exists in the specified Region`,
		secretID)
}

// Augment error message with extra details for most common exceptions:
func augmentErrMsg(secretID string, err error) string {
	if secretID == "" {
		logger.Warn("augmentErrMsg: SecretID is empty (which is unexpected)")
	}

	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case secretsmanager.ErrCodeResourceNotFoundException:
			secretID = "'" + secretID + "' "
			return resourceInitializationErrMsg(secretID)
		default:
			return fmt.Sprintf("secret %s: %s", secretID, err.Error())
		}
	} else {
		return fmt.Sprintf("secret %s: %s", secretID, err.Error())
	}
}

// GetDockerAuthFromASM makes the api call to the AWS Secrets Manager service to
// retrieve the docker auth data
func GetDockerAuthFromASM(secretID string, client secretsmanageriface.SecretsManagerAPI) (registry.AuthConfig, error) {
	in := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	}

	out, err := client.GetSecretValue(in)
	if err != nil {
		return registry.AuthConfig{}, errors.Wrapf(err,
			"asm fetching secret from the service for %s", secretID)
	}

	return extractASMValue(out)
}

func extractASMValue(out *secretsmanager.GetSecretValueOutput) (registry.AuthConfig, error) {
	if out == nil {
		return registry.AuthConfig{}, errors.New(
			"asm fetching authorization data: empty response")
	}

	secretValue := aws.StringValue(out.SecretString)
	if secretValue == "" {
		return registry.AuthConfig{}, errors.New(
			"asm fetching authorization data: empty secrets value")
	}

	authDataValue := AuthDataValue{}
	err := json.Unmarshal([]byte(secretValue), &authDataValue)
	if err != nil {
		// could  not unmarshal, incorrect secret value schema
		return registry.AuthConfig{}, errors.New(
			"asm fetching authorization data: unable to unmarshal secret value, invalid schema")
	}

	username := aws.StringValue(authDataValue.Username)
	password := aws.StringValue(authDataValue.Password)

	if username == "" {
		return registry.AuthConfig{}, errors.New(
			"asm fetching username: AuthorizationData is malformed, empty field")
	}

	if password == "" {
		return registry.AuthConfig{}, errors.New(
			"asm fetching password: AuthorizationData is malformed, empty field")
	}

	dac := registry.AuthConfig{
		Username: username,
		Password: password,
	}

	return dac, nil
}

func GetSecretFromASMWithInput(input *secretsmanager.GetSecretValueInput,
	client secretsmanageriface.SecretsManagerAPI, jsonKey string) (string, error) {
	secretID := *input.SecretId
	out, err := client.GetSecretValue(input)
	if err != nil {
		return "", errors.Wrap(err, augmentErrMsg(secretID, err))
	}

	if jsonKey == "" {
		return aws.StringValue(out.SecretString), nil
	}

	secretMap := make(map[string]interface{})
	jsonErr := json.Unmarshal([]byte(*out.SecretString), &secretMap)
	if jsonErr != nil {
		seelog.Warnf("Error when treating retrieved secret value with secret id %s as JSON and calling unmarshal.", *input.SecretId)
		return "", jsonErr
	}

	secretValue, ok := secretMap[jsonKey]
	if !ok {
		err = errors.New(fmt.Sprintf("retrieved secret from Secrets Manager did not contain json key %s", jsonKey))
		return "", err
	}

	return fmt.Sprintf("%v", secretValue), nil
}

// GetSecretFromASM makes the api call to the AWS Secrets Manager service to
// retrieve the secret value
func GetSecretFromASM(secretID string, client secretsmanageriface.SecretsManagerAPI) (string, error) {
	in := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	}

	out, err := client.GetSecretValue(in)
	if err != nil {
		return "", errors.Wrapf(err, "secret %s", secretID)
	}

	return aws.StringValue(out.SecretString), nil
}
