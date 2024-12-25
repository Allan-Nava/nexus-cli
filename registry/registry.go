package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/tidwall/gjson"
)


var      Start = time.Now()
var elapsed = time.Since(Start)
const ACCEPT_HEADER_V1 = "application/vnd.docker.distribution.manifest.v1+json"
const ACCEPT_HEADER = "application/vnd.docker.distribution.manifest.v2+json"
const CREDENTIALS_FILE = ".credentials"

type Registry struct {
	Host       string `toml:"nexus_host"`
	Username   string `toml:"nexus_username"`
	Password   string `toml:"nexus_password"`
	Repository string `toml:"nexus_repository"`
}

type Repositories struct {
	Images []string `json:"repositories"`
}

type ImageTags struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type ImageManifest struct {
	SchemaVersion int64       `json:"schemaVersion"`
	MediaType     string      `json:"mediaType"`
	Config        LayerInfo   `json:"config"`
	Layers        []LayerInfo `json:"layers"`
}
type LayerInfo struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

type ImageManifestV1 struct {
	SchemaVersion int64  `json:"schemaVersion"`
	Name          string `json:"name"`
	Tag           string `json:"tag"`
	Architecture  string `json:"architecture"`
	Created       string
	Date          time.Time
}

func NewRegistry() (Registry, error) {
	r := Registry{}
	if _, err := os.Stat(CREDENTIALS_FILE); os.IsNotExist(err) {
		return r, errors.New(fmt.Sprintf("%s file not found\n", CREDENTIALS_FILE))
	} else if err != nil {
		return r, err
	}

	if _, err := toml.DecodeFile(CREDENTIALS_FILE, &r); err != nil {
		return r, err
	}
	return r, nil
}

func (r Registry) ListImages() ([]string, error) {
	client := &http.Client{}

	url := fmt.Sprintf("%s/repository/%s/v2/_catalog", r.Host, r.Repository)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(r.Username, r.Password)
	req.Header.Add("Accept", ACCEPT_HEADER)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("HTTP Code: %d", resp.StatusCode))
	}

	var repositories Repositories
	json.NewDecoder(resp.Body).Decode(&repositories)

	return repositories.Images, nil
}

func (r Registry) ListTagsByImage(image string) ([]string, error) {
	client := &http.Client{}

	url := fmt.Sprintf("%s/repository/%s/v2/%s/tags/list", r.Host, r.Repository, image)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(r.Username, r.Password)
	req.Header.Add("Accept", ACCEPT_HEADER)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("HTTP Code: %d", resp.StatusCode))
	}

	var imageTags ImageTags
	json.NewDecoder(resp.Body).Decode(&imageTags)

	return imageTags.Tags, nil
}

func (r Registry) ImageManifest(image string, tag string) (ImageManifest, error) {
	var imageManifest ImageManifest
	client := &http.Client{}

	url := fmt.Sprintf("%s/repository/%s/v2/%s/manifests/%s", r.Host, r.Repository, image, tag)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return imageManifest, err
	}
	req.SetBasicAuth(r.Username, r.Password)
	req.Header.Add("Accept", ACCEPT_HEADER)

	resp, err := client.Do(req)
	if err != nil {
		return imageManifest, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return imageManifest, errors.New(fmt.Sprintf("HTTP Code: %d", resp.StatusCode))
	}

	json.NewDecoder(resp.Body).Decode(&imageManifest)

	return imageManifest, nil

}

func (r Registry) ImageManifestV1(image string, tag string) (ImageManifestV1, error) {
    var tr = &http.Transport{
            MaxIdleConnsPerHost: 90,
    }
	var imageManifest ImageManifestV1
	var client = &http.Client{
            Transport: tr,
    }
//elapsed = time.Since(Start)
//  log.Printf("tag is   %s", tag)
//  log.Printf("begin get tag %s", elapsed)	
//   Start = time.Now()
	url := fmt.Sprintf("%s/repository/%s/v2/%s/manifests/%s", r.Host, r.Repository, image, tag)
	req, err := http.NewRequest("GET", url, nil)
	req.Header = http.Header{
    "Accept": {"application/vnd.docker.distribution.manifest.v2+json"},
}
	if err != nil {
		return imageManifest, err
	}
	req.SetBasicAuth(r.Username, r.Password)
	req.Header.Add("Accept", ACCEPT_HEADER_V1)

	resp, err := client.Do(req)
	if err != nil {
		return imageManifest, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return imageManifest, errors.New(fmt.Sprintf("HTTP Code: %d", resp.StatusCode))
	}
	b, err := io.ReadAll(resp.Body)
	// b, err := ioutil.ReadAll(resp.Body)  Go.1.15 and earlier
	if err != nil {
		log.Fatalln(err)
	}
        resp = nil
	//json.NewDecoder(resp.Body).Decode(&imageManifest)
	compatibilityString := gjson.GetBytes(b, `config`)
	b = nil 
	digest := gjson.Get(compatibilityString.String(), `digest`)

// log.Printf("digest is    %s", digest)

        url = fmt.Sprintf("%s/repository/%s/v2/%s/blobs/%s", r.Host, r.Repository, image, digest)
        req, err = http.NewRequest("GET", url, nil)
        req.Header = http.Header{
    "Accept": {"application/vnd.docker.distribution.manifest.v2+json"},
}
        if err != nil {
                return imageManifest, err
        }
        req.SetBasicAuth(r.Username, r.Password)
        req.Header.Add("Accept", ACCEPT_HEADER_V1)

        resp, err = client.Do(req)
        if err != nil {
                return imageManifest, err
        }
        defer resp.Body.Close()
        if resp.StatusCode != 200 {
                return imageManifest, errors.New(fmt.Sprintf("HTTP Code: %d", resp.StatusCode))
        }
        b, err = io.ReadAll(resp.Body)
        // b, err := ioutil.ReadAll(resp.Body)  Go.1.15 and earlier
        if err != nil {
                log.Fatalln(err)
        }
        resp = nil
        //json.NewDecoder(resp.Body).Decode(&imageManifest)
	created := gjson.Get(string(b),"created")
        b = nil
if !created.Exists() {
                return imageManifest, err
}

// log.Printf("created  is   %s", created.String())
	imageManifest.Created = created.String()
	imageManifest.Date = created.Time()
	imageManifest.Tag = tag
	imageManifest.Name = image
	//
	return imageManifest, nil
}

func (r Registry) DeleteImageByTag(image string, tag string) error {
	sha, err := r.getImageSHA(image, tag)
	if err != nil {
		return err
	}
	client := &http.Client{}

	url := fmt.Sprintf("%s/repository/%s/v2/%s/manifests/%s", r.Host, r.Repository, image, sha)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(r.Username, r.Password)
	req.Header.Add("Accept", ACCEPT_HEADER)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		return errors.New(fmt.Sprintf("HTTP Code: %d", resp.StatusCode))
	}

	fmt.Printf("%s:%s has been successful deleted\n", image, tag)

	return nil
}

func (r Registry) getImageSHA(image string, tag string) (string, error) {
	client := &http.Client{}

	url := fmt.Sprintf("%s/repository/%s/v2/%s/manifests/%s", r.Host, r.Repository, image, tag)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(r.Username, r.Password)
	req.Header.Add("Accept", ACCEPT_HEADER)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New(fmt.Sprintf("HTTP Code: %d", resp.StatusCode))
	}

	return resp.Header.Get("docker-content-digest"), nil
}