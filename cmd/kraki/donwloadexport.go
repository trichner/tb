package kraki

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"regexp"
	"time"

	"cloud.google.com/go/storage"

	"github.com/trichner/oauthflows"
	vault2 "github.com/trichner/tb/pkg/vault"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

const bytesPerMiB = 1024 * 1024

type writeCounter struct {
	Sum        int64
	Total      int64
	LastUpdate time.Time
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)

	now := time.Now()
	if now.Before(wc.LastUpdate.Add(time.Second * 3)) {
		return n, nil
	}

	wc.Sum += int64(n)
	percent := float64(wc.Sum) / float64(wc.Total) * 100.0
	log.Printf("Read %3.1f%% (%dMiB/%dMiB)", percent, wc.Sum/bytesPerMiB, wc.Total/bytesPerMiB)
	wc.LastUpdate = time.Now()
	return n, nil
}

func downloadExport(matterId string) error {
	ctx := context.Background()

	config, err := getOAuth2Config()
	if err != nil {
		return err
	}
	config.Scopes = scopes

	tokenSource, err := oauthflows.NewBrowserFlowTokenSource(ctx, config)
	if err != nil {
		return err
	}

	svc, err := vault2.NewService(ctx, tokenSource)
	if err != nil {
		return err
	}

	matter, err := svc.FindMatter(ctx, matterId)
	if err != nil {
		return err
	}

	exports, err := svc.FindExportsByMatterId(ctx, matterId)
	if err != nil {
		return err
	}

	dir := sanitizeFileName(matter.Name)
	err = os.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to prepare directory %q: %w", dir, err)
	}

	for _, e := range exports {
		if e.Status != vault2.ExportStatusCompleted {
			log.Printf("export %q (%s) not completed: %q", e.Name, e.Id, e.Status.String())
		}
		for _, f := range e.CloudStorageSink.Files {
			filename := path.Base(f.ObjectName)
			dst := path.Join(dir, filename)
			log.Printf("dowloading export %q", f.ObjectName)
			ok, err := validateExistingFile(dst, f.Md5Hash)
			if err != nil {
				return fmt.Errorf("cannot validate %q: %w", dst, err)
			}
			if ok {
				log.Printf("exported file already downloaded as %q", filename)
				continue
			}

			log.Printf("transferring %q", filename)
			err = transferObjectToFile(ctx, tokenSource, dst, f)
			if err != nil {
				return fmt.Errorf("failed to download export %q (%s) : %w", e.Name, e.Id, err)
			}

		}
	}
	return nil
}

func validateExistingFile(dst string, md5sum string) (bool, error) {
	f, err := os.OpenFile(dst, os.O_RDONLY, 0)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	hasher := md5.New()

	if _, err = io.Copy(hasher, f); err != nil {
		return false, fmt.Errorf("cannot calculate md5sum of %q: %w", dst, err)
	}

	calculatedSum := hasher.Sum(nil)

	expectedSum, err := hex.DecodeString(md5sum)
	if err != nil {
		return false, fmt.Errorf("invalid md5sum provided %q: %w", md5sum, err)
	}

	return bytes.Equal(calculatedSum, expectedSum), nil
}

func transferObjectToFile(ctx context.Context, source oauth2.TokenSource, dst string, src *vault2.CloudStorageFile) error {
	file, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("failed to prepare file %q: %w", src.ObjectName, err)
	}
	defer file.Close()

	err = downloadFile(ctx, source, file, src.Size, src.BucketName, src.ObjectName)
	if err != nil {
		return fmt.Errorf("failed to download file %q: %w", src.ObjectName, err)
	}
	return nil
}

func sanitizeFileName(name string) string {
	reg := regexp.MustCompile(`[^A-Za-z0-9-_.@]*`)
	res := reg.ReplaceAllString(name, "")
	return res
}

func downloadFile(ctx context.Context, source oauth2.TokenSource, w io.Writer, totalBytes int64, bucket, object string) error {
	client, err := storage.NewClient(ctx, option.WithTokenSource(source))
	if err != nil {
		return fmt.Errorf("storage.NewClient: %w", err)
	}
	defer client.Close()

	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("Object(%q).NewReader: %w", object, err)
	}
	defer rc.Close()

	src := io.TeeReader(rc, &writeCounter{Total: totalBytes})
	_, err = io.Copy(w, src)
	if err != nil {
		return fmt.Errorf("io.Copy: %w", err)
	}
	return nil
}
