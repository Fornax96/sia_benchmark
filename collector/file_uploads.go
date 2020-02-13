package collector

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/Fornaxian/log"
	"gitlab.com/NebulousLabs/Sia/modules"
	sia "gitlab.com/NebulousLabs/Sia/node/api/client"
	"gitlab.com/NebulousLabs/fastrand"
	"lukechampine.com/frand"
)

func newSiaPath(name string) (siaPath modules.SiaPath) {
	siaPath, err := modules.NewSiaPath(
		string(name[0]) + "/" + string(name[1]) + "/" + name,
	)
	if err != nil {
		panic(err)
	}
	return siaPath
}

// UploadFile generates a new file of configurable size at the given path and
// uploads it to Sia
func UploadFile(
	sc *sia.Client,
	dir string,
	dataPieces, parityPieces uint64,
	size uint64,
) (err error) {
	var name = hex.EncodeToString(fastrand.Bytes(16)) + ".dat"
	var localPath = dir + "/" + name

	file, err := os.Create(localPath)
	if err != nil {
		return err
	}

	_, err = io.CopyN(file, frand.Reader, int64(size))
	file.Close()
	if err != nil {
		os.Remove(localPath) // Clean up on error
		return err
	}

	// We have a file of `size` bytes at `path`. Now upload it to Sia

	if err = sc.RenterUploadPost(
		dir+"/"+name,
		newSiaPath(name),
		dataPieces,
		parityPieces,
	); err != nil {
		os.Remove(localPath) // Clean up on error
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
		siafile, err := sc.RenterFileGet(newSiaPath(file.Name()))
		if err != nil {
			return fmt.Errorf("error getting '%s' from Sia: %s", file.Name(), err)
		}

		if siafile.File.UploadProgress >= 100 && siafile.File.MaxHealthPercent >= 100 {
			log.Debug("File '%s' is done uploading, removing local copy", file.Name())
			// Upload is done, remove source file
			if err = os.Remove(uploadsDir + "/" + file.Name()); err != nil {
				return fmt.Errorf("error removing '%s': %s", uploadsDir+"/"+file.Name(), err)
			}
		}
	}
	return nil
}
