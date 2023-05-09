package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOCIRepository(t *testing.T) {

	ctx := context.Background()
	repo := newOCIKameletRepository("docker.io/lburgazzoli/camel-kamelets:latest")

	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.True(t, len(list) > 0)
	require.Contains(t, list, "aws-s3-sink")

	k, err := repo.Get(ctx, "aws-s3-sink")
	require.NoError(t, err)
	require.NotNil(t, k)
	require.Equal(t, "aws-s3-sink", k.Name)

}

func TestInvalidOCIRepository(t *testing.T) {
	image := "docker.io/foo/bar"
	ctx := context.Background()
	repo := newOCIKameletRepository(image)

	list, err := repo.List(ctx)
	require.Nil(t, list)
	require.EqualError(t, err, fmt.Sprintf("unable to determine image name and/or tag from %s", image))
}
