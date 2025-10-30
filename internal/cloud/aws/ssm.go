package aws

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/estudosdevops/opsmaster/internal/cloud"
	"github.com/estudosdevops/opsmaster/internal/logger"
)

const (
	// connectivityTestTimeout is the maximum time to wait for network connectivity test commands.
	// Network operations may take longer than regular commands due to connection attempts.
	connectivityTestTimeout = 30 * time.Second
)

// AWSProvider implements cloud.CloudProvider interface for AWS.
// Uses AWS Systems Manager (SSM) for remote command execution and
// EC2 API for instance tagging.
//
// This is the concrete implementation of the CloudProvider abstraction,
// allowing the rest of the codebase to work with AWS without knowing
// AWS-specific details.
type AWSProvider struct {
	sessionManager *SessionManager
	log            *slog.Logger
}

// NewAWSProvider creates a new AWS provider with connection pooling
func NewAWSProvider() *AWSProvider {
	return &AWSProvider{
		sessionManager: NewSessionManager(),
		log:            logger.Get(),
	}
}

// NewAWSProviderWithProfile creates a new AWS provider with specific AWS profile.
// This enables SSO authentication by using named profiles from ~/.aws/config.
//
// Parameters:
//   - ctx: Context for timeout and cancellation control
//   - profile: AWS profile name (e.g., "sso-production", "dev-account")
//
// Returns:
//   - *AWSProvider: Configured provider instance
//   - error: Error if profile is invalid or AWS config fails
//
// Example usage:
//
//	provider, err := NewAWSProviderWithProfile(ctx, "sso-production")
//	if err != nil {
//	    return fmt.Errorf("failed to create AWS provider: %w", err)
//	}
//
// SSO Flow:
//  1. Reads profile from ~/.aws/config
//  2. If SSO configured, uses cached credentials or prompts for login
//  3. Creates provider with authenticated session
func NewAWSProviderWithProfile(ctx context.Context, profile string) (*AWSProvider, error) {
	// Validate profile parameter
	if profile == "" {
		return nil, fmt.Errorf("profile cannot be empty")
	}

	// Create session manager with profile support
	sessionManager, err := NewSessionManagerWithProfile(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager with profile %s: %w", profile, err)
	}

	return &AWSProvider{
		sessionManager: sessionManager,
		log:            logger.Get(),
	}, nil
}

// Name returns the provider name
func (*AWSProvider) Name() string {
	return "aws"
}

// ValidateInstance checks if instance is accessible via SSM.
// An instance must be:
// 1. Registered in SSM
// 2. Online (ping status = Online)
// 3. SSM agent running and healthy
//
// Returns error if instance is not reachable via SSM.
func (p *AWSProvider) ValidateInstance(ctx context.Context, instance *cloud.Instance) error {
	p.log.Debug("Validating instance SSM connectivity",
		"instance_id", instance.ID,
		"account", instance.Account,
		"region", instance.Region)

	// Get SSM client for this instance's profile/region
	profile := getProfileForInstance(instance)
	client, err := p.sessionManager.GetSSMClient(ctx, profile, instance.Region)
	if err != nil {
		return fmt.Errorf("failed to get SSM client: %w", err)
	}

	// Query instance information from SSM
	input := &ssm.DescribeInstanceInformationInput{
		Filters: []types.InstanceInformationStringFilter{
			{
				Key:    aws.String("InstanceIds"),
				Values: []string{instance.ID},
			},
		},
	}

	output, err := client.DescribeInstanceInformation(ctx, input)
	if err != nil {
		return fmt.Errorf("SSM API error for instance %s: %w", instance.ID, err)
	}

	// Check if instance was found
	if len(output.InstanceInformationList) == 0 {
		return fmt.Errorf("instance %s not found in SSM - ensure SSM agent is installed and running", instance.ID)
	}

	// Check ping status (must be Online)
	info := output.InstanceInformationList[0]
	if info.PingStatus != types.PingStatusOnline {
		return fmt.Errorf("instance %s is %s (expected Online) - SSM agent may be stopped or network issue",
			instance.ID, info.PingStatus)
	}

	p.log.Debug("Instance SSM validation successful",
		"instance_id", instance.ID,
		"ping_status", info.PingStatus,
		"platform", info.PlatformType)

	return nil
}

// ExecuteCommand executes shell commands remotely on the instance via SSM.
// Uses AWS-RunShellScript document to execute commands.
//
// Parameters:
//   - ctx: context for timeout/cancellation
//   - instance: target instance
//   - commands: slice of shell commands to execute
//   - timeout: maximum execution time
//
// Returns CommandResult with stdout, stderr, exit code, and duration.
func (p *AWSProvider) ExecuteCommand(ctx context.Context, instance *cloud.Instance, commands []string, timeout time.Duration) (*cloud.CommandResult, error) {
	p.log.Info("Executing commands on instance",
		"instance_id", instance.ID,
		"commands_count", len(commands),
		"timeout", timeout)

	// Get SSM client
	profile := getProfileForInstance(instance)
	client, err := p.sessionManager.GetSSMClient(ctx, profile, instance.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSM client: %w", err)
	}

	// Send command via SSM
	sendInput := &ssm.SendCommandInput{
		InstanceIds:  []string{instance.ID},
		DocumentName: aws.String("AWS-RunShellScript"),
		Parameters: map[string][]string{
			"commands": commands,
		},
		TimeoutSeconds: aws.Int32(int32(timeout.Seconds())),
		Comment:        aws.String("OpsMaster package installation"),
	}

	sendOutput, err := client.SendCommand(ctx, sendInput)
	if err != nil {
		return nil, fmt.Errorf("failed to send SSM command: %w", err)
	}

	commandID := *sendOutput.Command.CommandId
	p.log.Debug("SSM command sent",
		"instance_id", instance.ID,
		"command_id", commandID)

	// Wait for command completion and get result
	return p.waitForCommand(ctx, client, commandID, instance.ID, timeout)
}

// TestConnectivity tests network connectivity from instance to a host:port.
// Uses multiple methods for better compatibility across different OS distributions.
//
// Methods tried in order:
// 1. nc (netcat) - most reliable
// 2. telnet - fallback
// 3. /dev/tcp - bash built-in (limited compatibility)
//
// This is useful for validating prerequisites, e.g., checking if instance
// can reach Puppet Server before attempting installation.
func (p *AWSProvider) TestConnectivity(ctx context.Context, instance *cloud.Instance, host string, port int) error {
	p.log.Info("Testing connectivity",
		"instance_id", instance.ID,
		"target", fmt.Sprintf("%s:%d", host, port))

	// Script that tries multiple methods with fallback
	testScript := fmt.Sprintf(`#!/bin/bash
set +e  # Don't exit on error

TARGET_HOST="%s"
TARGET_PORT="%d"

echo "=== Testing connectivity to ${TARGET_HOST}:${TARGET_PORT} ==="

# Method 1: Try nc (netcat) first
if command -v nc >/dev/null 2>&1; then
    echo "Trying nc (netcat)..."
    timeout 10 nc -zv "${TARGET_HOST}" "${TARGET_PORT}" 2>&1
    if [ $? -eq 0 ]; then
        echo "SUCCESS: nc test passed"
        exit 0
    fi
    echo "nc failed, trying next method..."
fi

# Method 2: Try telnet
if command -v telnet >/dev/null 2>&1; then
    echo "Trying telnet..."
    timeout 10 bash -c "echo -e '\x1dclose\x0d' | telnet ${TARGET_HOST} ${TARGET_PORT}" 2>&1 | grep -q "Connected\|Escape"
    if [ $? -eq 0 ]; then
        echo "SUCCESS: telnet test passed"
        exit 0
    fi
    echo "telnet failed, trying next method..."
fi

# Method 3: Try /dev/tcp (bash built-in)
echo "Trying /dev/tcp..."
timeout 10 bash -c "cat < /dev/null > /dev/tcp/${TARGET_HOST}/${TARGET_PORT}" 2>&1
if [ $? -eq 0 ]; then
    echo "SUCCESS: /dev/tcp test passed"
    exit 0
fi

# All methods failed
echo "FAILED: All connectivity test methods failed"
echo "Error details:"
echo "  - nc: not available or connection failed"
echo "  - telnet: not available or connection failed"
echo "  - /dev/tcp: not available or connection failed"
exit 1
`, host, port)

	commands := []string{testScript}

	// Increase timeout for network operations (connectivity tests may take longer)
	result, err := p.ExecuteCommand(ctx, instance, commands, connectivityTestTimeout)
	if err != nil {
		return fmt.Errorf("connectivity test execution failed: %w", err)
	}

	// Check if any method succeeded
	if strings.Contains(result.Stdout, "SUCCESS") {
		p.log.Info("Connectivity test passed",
			"instance_id", instance.ID,
			"target", fmt.Sprintf("%s:%d", host, port))
		return nil
	}

	// Test failed - provide detailed error
	p.log.Error("Connectivity test failed",
		"instance_id", instance.ID,
		"target", fmt.Sprintf("%s:%d", host, port),
		"output", result.Stdout,
		"error", result.Stderr)

	return fmt.Errorf("cannot reach %s:%d from instance %s\nOutput:\n%s\nError:\n%s",
		host, port, instance.ID, result.Stdout, result.Stderr)
}

// waitForCommand polls SSM until command completes or times out.
// Uses exponential backoff polling pattern.
//
// SSM commands are asynchronous - SendCommand returns immediately,
// then we must poll GetCommandInvocation to get the result.
func (p *AWSProvider) waitForCommand(ctx context.Context, client *ssm.Client, commandID, instanceID string, timeout time.Duration) (*cloud.CommandResult, error) {
	start := time.Now()
	ticker := time.NewTicker(2 * time.Second) // Poll every 2 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("command canceled: %w", ctx.Err())

		case <-time.After(timeout):
			return nil, fmt.Errorf("command timeout after %v", timeout)

		case <-ticker.C:
			// Query command status
			input := &ssm.GetCommandInvocationInput{
				CommandId:  aws.String(commandID),
				InstanceId: aws.String(instanceID),
			}

			output, err := client.GetCommandInvocation(ctx, input)
			if err != nil {
				// Command might not be ready yet, continue polling
				continue
			}

			// Check if command finished (success or failure)
			if output.Status == types.CommandInvocationStatusSuccess ||
				output.Status == types.CommandInvocationStatusFailed ||
				output.Status == types.CommandInvocationStatusTimedOut ||
				output.Status == types.CommandInvocationStatusCancelled {
				// Build result
				result := &cloud.CommandResult{
					InstanceID: instanceID,
					ExitCode:   int(output.ResponseCode),
					Stdout:     aws.ToString(output.StandardOutputContent),
					Stderr:     aws.ToString(output.StandardErrorContent),
					Duration:   time.Since(start),
				}

				// Set error if command failed
				if output.Status != types.CommandInvocationStatusSuccess {
					result.Error = fmt.Errorf("command %s with exit code %d", output.Status, result.ExitCode)
				}

				p.log.Debug("Command completed",
					"instance_id", instanceID,
					"command_id", commandID,
					"status", output.Status,
					"exit_code", result.ExitCode,
					"duration", result.Duration)

				return result, nil
			}

			// Command still running (InProgress, Pending), continue polling
			p.log.Debug("Command still running",
				"instance_id", instanceID,
				"command_id", commandID,
				"status", output.Status,
				"elapsed", time.Since(start))
		}
	}
}

// TagInstance adds tags to an EC2 instance.
// Tags are used to mark instances after successful installation.
//
// Common use cases:
//   - Mark instances as "puppet=true" after Puppet installation
//   - Add timestamp tags for audit trail
//   - Tag with installer metadata
//
// Note: Tags are applied at EC2 level, not SSM. Requires ec2:CreateTags permission.
func (p *AWSProvider) TagInstance(ctx context.Context, instance *cloud.Instance, tags map[string]string) error {
	p.log.Info("Tagging instance",
		"instance_id", instance.ID,
		"tags_count", len(tags))

	// Get EC2 client (not SSM, as tags are EC2 resources)
	profile := getProfileForInstance(instance)
	ec2Client, err := p.sessionManager.GetEC2Client(ctx, profile, instance.Region)
	if err != nil {
		return fmt.Errorf("failed to get EC2 client: %w", err)
	}

	// Convert map to EC2 tag slice
	var ec2Tags []ec2types.Tag
	for key, value := range tags {
		ec2Tags = append(ec2Tags, ec2types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	// Create tags on instance
	input := &ec2.CreateTagsInput{
		Resources: []string{instance.ID},
		Tags:      ec2Tags,
	}

	_, err = ec2Client.CreateTags(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to tag instance %s: %w", instance.ID, err)
	}

	p.log.Info("Instance tagged successfully",
		"instance_id", instance.ID,
		"tags", tags)

	return nil
}

// HasTag checks if instance already has a specific tag with given value.
// Useful for idempotency - skip processing if instance already tagged.
//
// Example: Check if instance has "puppet=true" before reinstalling Puppet.
//
// Returns true if tag exists with exact key and value, false otherwise.
func (p *AWSProvider) HasTag(ctx context.Context, instance *cloud.Instance, key, value string) (bool, error) {
	p.log.Debug("Checking instance tag",
		"instance_id", instance.ID,
		"tag_key", key,
		"tag_value", value)

	// Get EC2 client
	profile := getProfileForInstance(instance)
	ec2Client, err := p.sessionManager.GetEC2Client(ctx, profile, instance.Region)
	if err != nil {
		return false, fmt.Errorf("failed to get EC2 client: %w", err)
	}

	// Query tags for this instance
	input := &ec2.DescribeTagsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("resource-id"),
				Values: []string{instance.ID},
			},
			{
				Name:   aws.String("key"),
				Values: []string{key},
			},
		},
	}

	output, err := ec2Client.DescribeTags(ctx, input)
	if err != nil {
		return false, fmt.Errorf("failed to describe tags for instance %s: %w", instance.ID, err)
	}

	// Check if tag with specified value exists
	for _, tag := range output.Tags {
		if aws.ToString(tag.Key) == key && aws.ToString(tag.Value) == value {
			p.log.Debug("Tag found on instance",
				"instance_id", instance.ID,
				"tag_key", key,
				"tag_value", value)
			return true, nil
		}
	}

	p.log.Debug("Tag not found on instance",
		"instance_id", instance.ID,
		"tag_key", key,
		"tag_value", value)

	return false, nil
}

// getProfileForInstance determines the AWS profile to use for a specific instance.
// Priority order:
//  1. aws_profile from instance metadata (from CSV)
//  2. Account ID as fallback (backward compatibility)
//
// This function extracts the correct AWS profile to use for authentication,
// ensuring that SSO profiles from CSV are used instead of account IDs.
func getProfileForInstance(instance *cloud.Instance) string {
	// Check if instance has aws_profile in metadata (from CSV)
	if profile := instance.Metadata["aws_profile"]; profile != "" {
		return profile
	}

	// Fallback to account ID for backward compatibility
	// Note: This might not work for SSO profiles, but maintains compatibility
	return instance.Account
}
