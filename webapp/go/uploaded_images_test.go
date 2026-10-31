package isureal

import (
	"os"
	"testing"
)

func TestRemoveUploadedImagesKeepsOtherFiles(t *testing.T) {
	dir := t.TempDir()
	oldDir := imageDiskCacheDir
	imageDiskCacheDir = dir
	defer func() { imageDiskCacheDir = oldDir }()

	uploaded := "0f8fad5b-d9cb-469f-a165-70867728950e"
	initial := "7c9e6679-7425-40de-944b-e07fc1f90ae7"
	for _, id := range []string{uploaded, initial} {
		path, _ := imageDiskPath(id)
		if err := writeImageFileDurable(path, []byte{0xff, 0xd8, 0xff, 0xd9}); err != nil {
			t.Fatal(err)
		}
	}
	if err := recordUploadedImage(uploaded); err != nil {
		t.Fatal(err)
	}
	if err := removeUploadedImages(); err != nil {
		t.Fatal(err)
	}
	if path, _ := imageDiskPath(uploaded); fileExists(path) {
		t.Error("uploaded image file was not removed")
	}
	if path, _ := imageDiskPath(initial); !fileExists(path) {
		t.Error("initial image file was removed")
	}
	if fileExists(uploadedImagesLog()) {
		t.Error("upload log was not removed")
	}
	// 2回目（ログがない）も失敗しない
	if err := removeUploadedImages(); err != nil {
		t.Fatal(err)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
