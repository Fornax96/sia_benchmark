package collector

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Fornaxian/log"
	"gitlab.com/NebulousLabs/Sia/modules"

	sia "gitlab.com/NebulousLabs/Sia/node/api/client"
	"gitlab.com/NebulousLabs/fastrand"
)

func newSiaPath(path string) (siaPath modules.SiaPath) {
	siaPath, err := modules.NewSiaPath(path)
	if err != nil {
		panic(err)
	}
	return siaPath
}

// UploadFile generates a new file of configurable size at the given path and
// uploads it to Sia
func UploadFile(
	sc *sia.Client,
	path string,
	dataPieces, parityPieces uint64,
	size uint64,
) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	_, err = io.CopyN(file, fastrand.Reader, int64(size))
	file.Close()
	if err != nil {
		os.Remove(path) // Clean up on error
		return err
	}

	// We have a file of `size` bytes at `path`. Now upload it to Sia

	err = sc.RenterUploadPost(
		path,
		newSiaPath("benchmark/"+filepath.Base(path)),
		dataPieces,
		parityPieces,
	)
	if err != nil {
		os.Remove(path) // Clean up on error
		return err
	}

	return nil
}

// FinishUploads looks through all the files in the uploads dir and removes the
// ones which have finished uploading to Sia
func FinishUploads(sc *sia.Client, uploadsDir string) error {
	files, err := ioutil.ReadDir(uploadsDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		siafile, err := sc.RenterFileGet(newSiaPath("benchmark/" + file.Name()))
		if err != nil {
			return err
		}

		if siafile.File.UploadProgress > 99.9 { // comparing floats.. dangerous
			log.Debug("File '%s' is done uploading, removing local copy", file.Name())
			// Upload is done, remove source file
			err = os.Remove(uploadsDir + "/" + file.Name())
			if err != nil {
				return err
			}
		}
	}
	return nil
}
