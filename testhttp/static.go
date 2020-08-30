package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/util"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
)

func (vfs StaticFS) prepare(name string) (*VFile, error) {

	log.Trace("check virtual file, ", name)

	name = path.Clean(name)

	if strings.HasSuffix(name, "/") {
		name = name + "index.html"
	}

	log.Trace("clean virtual file, ", name)

	f, present := data[name]
	if !present {
		log.Trace("virtual file not found, ", name)
		return nil, os.ErrNotExist
	}
	var err error
	vfs.once.Do(func() {
		f.FileName = path.Base(name)
		if f.FileSize == 0 {
			return
		}
		var gr *gzip.Reader
		b64 := base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(f.Compressed))
		gr, err = gzip.NewReader(b64)
		if err != nil {
			log.Error(err)
			return
		}
		f.Data, err = ioutil.ReadAll(gr)
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return f, nil
}

func (vfs StaticFS) Open(name string) (http.File, error) {

	name = path.Clean(name)

	if vfs.CheckLocalFirst {

		if vfs.TrimLeftPath != "" {
			name = util.TrimLeftStr(name, vfs.TrimLeftPath)
		}

		localFile := path.Join(vfs.StaticFolder, name)

		log.Trace("check local file, ", localFile)

		if util.FileExists(localFile) {

			f2, err := os.Open(localFile)
			if err == nil {
				return f2, err
			}
		}

		log.Debug("local file not found,", localFile)
	}

	if vfs.SkipVFS {
		log.Trace("file was not found on vfs, ", name)
		return nil, errors.New("file not found")
	}

	f, err := vfs.prepare(name)
	if err != nil {
		return nil, err
	}
	log.Trace(f.FileName, ",", f.ModifyTime, ",", f.FileSize, ",", f.Mode(), ",", f.Name())
	return f.File()
}

type StaticFS struct {
	once            sync.Once
	StaticFolder    string
	TrimLeftPath    string
	CheckLocalFirst bool
	SkipVFS         bool
}

var data = map[string]*VFile{

	"/main.go": {
		FileName:   "main.go",
		FileSize:   1896,
		ModifyTime: 1598232559,
		Compressed: `
H4sIAAAAAAAA/4xVwW7jNhA9k18x1aEgtwadAkWzayCHIlmjAZK0WO/mEgQFIw0dwhJpkFRir6F/L4aS
HSXrFM0hlsh5b948zlBrXa70EqHR1nFum7UPCQRnxcM2YSw4K5Y2PbYPqvTNdOmDrWs9Lb1LuEkFZ48p
rYNvEwYorDPWWRUfpyboBp99WE1LH3Cq13baBxGfwzQlGD37nCFuXUm/yTZYcMl52q4Rbi9swDL5sIWY
Qlsm2HFmIgAQWs1tjYttTNhw5nSDFGTdknd7OAWMkOe+WQeMEat9JMsU9jsCgHXp9984u/aVNduvtsH9
ymWc+7rCAAAP3tecswudNNDf3T151NPcvFZgWleCqGwYVSHhrzU6MdIqQRxKmQCG4IMkqQFTGxxUNigT
VUbRc0b+AvQjX5IY+JBLlUD/xXuc2RPaeWML+5APWn1BXWGg98zGWXfQ8fMeRuF93Ax61A0+9wvCKPJF
TjhjmWAGYCacdRNwtj6m9rz2keRmiaOq3wmnNJUNovStS3Q4EsTdvY+50Etn/BEDna3fTb9IOpFZ/81g
3sXTgQs5HOQYofbtcDSr/U6o3Fs/gGjzGOjaVwQalNLbCHryDoJ6WEigkVK5n18gee2bsxth1EvDT+Dk
aFddxgsbhMzd/0ryfjKO1rmNfZkYjC5x142RvOvjb+cLId8O8yjw1obU6nq+2HUvU71fG1p413H+pAM8
mQh3928vBtqpfbkCumHUdZtwM2j9gksbE4b5Qpj4VkPuAMKpK1+uhOSM6M9Ar9foKvFDnp2J3YQkKKXk
gPzm6h57cOcg/X9fBJwZH+CfCTzB7AyCdkvMhdIcml9zIG08qQOf5IxZkzfOzqh1c+zB+R7DGQ1393pM
fFSfQ7jx6fPGxkSqWZZN3wWRDWE+qusVzWDxjA/F5OT09FTmG+FgJQ2VLeeL3fCQ+2MGffzXYJsrNOlv
nR5nUEzz4vkjlqsrX+p6bkNMM0ihxU5yzljTbqi6bMwNPi8wPOF1u6HjoD31p3ZVjaKYFpPRAVJUELm1
JEUOH6eBqH8jOtG0G9ofPMwEV1SH+8NVmUUUs4+fPn4qJjB87tR5jTr0aYPoqeTI8Z9Gjq+1s6XAEGR2
u+P/BgAA//+C6pkNaAcAAA==
`,
	},
}
