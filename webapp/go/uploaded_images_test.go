package isureal

import (
	"os"
	"testing"
)

// ベンチ前に task clean-uploads がこの一覧のファイルだけを消す（Taskfile.yml）
func TestRecordUploadedImageAppendsID(t *testing.T) {
	dir := t.TempDir()
	oldDir := imageDiskCacheDir
	imageDiskCacheDir = dir
	defer func() { imageDiskCacheDir = oldDir }()

	ids := []string{"0f8fad5b-d9cb-469f-a165-70867728950e", "7c9e6679-7425-40de-944b-e07fc1f90ae7"}
	for _, id := range ids {
		if err := recordUploadedImage(id); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(uploadedImagesLog())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), ids[0]+"\n"+ids[1]+"\n"; got != want {
		t.Errorf("upload log = %q, want %q", got, want)
	}
}
