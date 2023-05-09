package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/containers/image/docker/reference"
	"io/ioutil"
	"path"
	"path/filepath"
	"sync"

	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	"os"
	"strings"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type ociKameletRepository struct {
	once     sync.Once
	image    string
	relative string

	kamelets map[string]string
	pollRoot string
	pollErr  error
}

func newOCIKameletRepository(image string) KameletRepository {
	imageName := image
	relativePath := ""

	items := strings.Split(image, "?")
	if len(items) == 2 {
		imageName = items[0]
		relativePath = items[1]
	}

	repo := ociKameletRepository{
		image:    imageName,
		relative: relativePath,
		kamelets: make(map[string]string, 0),
	}

	return &repo
}

// Enforce type
var _ KameletRepository = &ociKameletRepository{}

func (r *ociKameletRepository) List(ctx context.Context) ([]string, error) {

	r.pullAll(ctx)

	if r.pollErr != nil {
		return nil, r.pollErr
	}

	kamelets := make([]string, 0, len(r.kamelets))
	for k := range r.kamelets {
		kamelets = append(kamelets, k)
	}

	return kamelets, r.pollErr
}

func (r *ociKameletRepository) Get(ctx context.Context, name string) (*camelv1.Kamelet, error) {
	r.pullAll(ctx)

	filePath, ok := r.kamelets[name]
	if !ok {
		return nil, fmt.Errorf("cannot find kamelet %s", name)
	}

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
		content, err = yaml.ToJSON(content)
		if err != nil {
			return nil, err
		}
	}

	var kamelet camelv1.Kamelet
	if err := json.Unmarshal(content, &kamelet); err != nil {
		return nil, err
	}

	return &kamelet, nil
}

func (r *ociKameletRepository) String() string {
	return fmt.Sprintf("Image[%s]", r.image)
}

// Pull download an image from the given registry and copy the content to a local temporary folder.
func (r *ociKameletRepository) pull(ctx context.Context, image string) (string, error) {
	repo, err := reference.Parse(image)
	if err != nil {
		return "", err
	}

	nt, ok := repo.(reference.NamedTagged)
	if !ok {
		return "", fmt.Errorf("unable to determine image name and/or tag from %s", image)
	}

	or, err := remote.NewRepository(nt.Name())
	if err != nil {
		return "", err
	}

	or.Client = &auth.Client{
		Client: retry.DefaultClient,
		Cache:  auth.DefaultCache,
	}

	f, err := os.MkdirTemp("", "camel-")
	if err != nil {
		return "", err
	}

	store, err := file.New(f)
	if err != nil {
		return "", err
	}

	if _, err = oras.Copy(ctx, or, nt.Tag(), store, nt.Tag(), oras.DefaultCopyOptions); err != nil {
		return "", err
	}

	return f, nil
}

func (r *ociKameletRepository) pullAll(ctx context.Context) {
	r.once.Do(func() {
		r.pollRoot, r.pollErr = r.pull(ctx, r.image)
		if r.pollErr != nil {
			return
		}

		entries, err := ioutil.ReadDir(path.Join(r.pollRoot, r.relative))
		if err != nil {
			r.pollErr = err
			return
		}

		for i := range entries {
			if entries[i].IsDir() {
				continue
			}

			name := filepath.Base(entries[i].Name())
			if isKameletFileName(name) {
				r.kamelets[getKameletNameFromFile(name)] = path.Join(r.pollRoot, r.relative, entries[i].Name())
			}
		}
	})

}
