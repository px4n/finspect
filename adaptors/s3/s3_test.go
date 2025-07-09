package s3_test

import (
	"context"
	"testing"

	"github.com/px4n/finspect/adaptors/cloud"
	"github.com/px4n/finspect/adaptors/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS3Adaptor(t *testing.T) {
	// This test requires AWS credentials and a test bucket
	// Skip if not configured
	t.Skip("S3 integration test requires AWS credentials")

	adaptor := s3.New()
	require.NotNil(t, adaptor)

	ctx := context.Background()

	// Test Connect
	err := adaptor.Connect(ctx, cloud.AdaptorConfig{
		Provider: cloud.ProviderS3,
		Bucket:   "test-bucket",
		Region:   "us-east-1",
		Credentials: map[string]interface{}{
			"access_key": "test-access-key",
			"secret_key": "test-secret-key",
		},
	})
	assert.Error(t, err) // Should fail with invalid credentials
}

func TestS3AdaptorInterface(_ *testing.T) {
	// Verify that S3 adaptor implements Adaptor interface
	var _ cloud.Adaptor = &s3.Adaptor{}
}
