package kraki

import (
	"context"

	"github.com/trichner/oauthflows"
	vault2 "github.com/trichner/tb/pkg/vault"
)

func describeMatter(matterId string) error {
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

	mappedExports := []*export{}
	for _, e := range exports {
		mappedExports = append(mappedExports, mapExport(e))
	}

	m := &describedMatter{
		Id:      matter.Id,
		Name:    matter.Name,
		State:   matter.State,
		Exports: mappedExports,
	}

	return printJson(m)
}

type describedMatter struct {
	Id      string
	Name    string
	State   vault2.MatterState
	Exports []*export
}

type export struct {
	Id                    string
	Name                  string
	CreateTime            string
	Status                vault2.ExportStatus
	ExportedArtifactCount int64
	TotalArtifactCount    int64
	SizeInBytes           int64
	Files                 []*file
}

type file struct {
	BucketName string
	ObjectName string
	Size       int64
	Md5Hash    string
}

func mapExport(e *vault2.Export) *export {
	files := []*file{}
	for _, f := range e.CloudStorageSink.Files {
		files = append(files, mapFile(f))
	}

	return &export{
		Id:                    e.Id,
		Name:                  e.Name,
		CreateTime:            e.CreateTime,
		Status:                e.Status,
		ExportedArtifactCount: e.Statistics.ExportedArtifactCount,
		TotalArtifactCount:    e.Statistics.TotalArtifactCount,
		SizeInBytes:           e.Statistics.SizeInBytes,
		Files:                 files,
	}
}

func mapFile(s *vault2.CloudStorageFile) *file {
	return &file{
		BucketName: s.BucketName,
		ObjectName: s.ObjectName,
		Size:       s.Size,
		Md5Hash:    s.Md5Hash,
	}
}
