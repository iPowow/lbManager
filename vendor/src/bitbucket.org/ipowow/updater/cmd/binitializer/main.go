package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"bitbucket.org/ipowow/updater"
	"github.com/aquam8/mlab-ns2/gae/ns/digest"
	cr "github.com/coreroller/coreroller/backend/src/api"
	updatr "github.com/coreroller/coreroller/updaters/lib/go"
	"github.com/mgutz/logxi/v1"
)

const (
	originalVersion = "0.0.0"

	// CoreRoller user used to request package version
	username = "binitializer"
	password = "rob0000t"
)

var (
	logger = log.New("binitializer")

	// ErrMissingCorerollerApiURL is the error returned when CR_API_URL is not provided
	ErrMissingCorerollerApiURL = errors.New("CR_API_URL environment variable is missing")

	// ErrMissingAppID is the error returned when xxx_CR_APP_ID is not provided
	ErrMissingAppID = errors.New("<artifact>_CR_APP_ID environment variable is missing")

	// ErrMissingGroupID is the error returned when xxx_CR_GROUP_ID is not provided
	ErrMissingGroupID = errors.New("<artifact>_CR_GROUP_ID environment variable is missing")
)

var p struct {
	artifacts     string
	executableDir string
	version       string
	crApiURL      string
}

func init() {
	flag.StringVar(&p.artifacts, "artifacts", "", "List of artifacts to be download and install (comma separated, no spaces)")
	flag.StringVar(&p.executableDir, "dir", "", "Output directory of where to install the binary(ies)")
	flag.StringVar(&p.version, "version", "", "A particular version you want to use: leave blank to get the channel's active version; use 'latest' to get the latest build; else use an existing version such as '0.0.3'")
}

func main() {
	flag.Parse()

	if err := validateParams(); err != nil {
		logger.Fatal("Invalid parameters", "err", err)
	}

	artifacts := parseArtifacts(p.artifacts)
	fetchAndInstall(artifacts)
}

func validateParams() error {
	if p.artifacts == "" {
		return errors.New("no artifacts to deploy provided")
	}

	if p.executableDir == "" {
		return errors.New("no output directory provided")
	}

	return nil
}

func getContextFromEnvironment(prefix string) (crApiURL string, appID string, groupID string, err error) {

	crApiURL = os.Getenv("CR_API_URL")
	if crApiURL == "" {
		err = ErrMissingCorerollerApiURL
		return
	}

	appID = os.Getenv(strings.ToUpper(prefix) + "_CR_APP_ID")
	if appID == "" {
		err = ErrMissingAppID
		return
	}

	groupID = os.Getenv(strings.ToUpper(prefix) + "_CR_GROUP_ID")
	if groupID == "" {
		err = ErrMissingGroupID
		return
	}
	return
}

func parseArtifacts(artifactsStr string) []string {
	artifacts := make([]string, 0)

	for _, artifact := range strings.Split(artifactsStr, ",") {
		artifacts = append(artifacts, artifact)
	}

	return artifacts
}

func fetchAndInstall(artifacts []string) {
	for _, artifactStr := range artifacts {
		logger.Info("fetching artifact", "artifact", artifactStr)

		// If a version is specifed we bypass CoreRoller and ask to download a crafted package
		if p.version != "" {

			if err := downloadVersionedPackage(artifactStr); err != nil {
				logger.Error("cannot craft package request", "artifact", artifactStr, "error", err.Error())
				continue
			}

		} else {

			a, err := createArtifactObject(artifactStr)
			if err != nil {
				logger.Error("cannot get context configuration", "artifact", artifactStr, "error", err.Error())
				continue
			}

			pkg, err := getLatestPackageFromCoreroller(a)
			if err != nil {
				logger.Error("failed to fetch coreroller latest package", "artifact", artifactStr, "error", err.Error())
				continue
			}

			update, err := buildUpdateObject(pkg)
			if err != nil {
				logger.Error("failed to install coreroller package", "artifact", artifactStr, "error", err.Error())
				continue
			}

			if err := processUpdate(a, update); err != nil {
				logger.Error("failed to install coreroller package", "artifact", artifactStr, "error", err.Error())
				continue
			}
		}
	}
}

func downloadVersionedPackage(artifactStr string) error {
	a := &updater.Artifact{
		ExecutableDir:    p.executableDir,
		ExecutablePrefix: artifactStr,
		Version:          originalVersion,
	}

	if err := a.Validate(); err != nil {
		return err
	}

	URL := a.PackageURL()
	filename := a.VersionedArtifact(p.version)
	update := &updatr.Update{
		Version:  p.version,
		URL:      URL,
		Filename: filename,
	}

	if err := processUpdate(a, update); err != nil {
		logger.Error("failed to install package", "artifact", artifactStr, "version", p.version, "filename", filename, "URL", URL, "error", err.Error())
		return err
	}

	return nil
}

func createArtifactObject(artifact string) (*updater.Artifact, error) {

	a := &updater.Artifact{
		ExecutableDir:    p.executableDir,
		ExecutablePrefix: artifact,
		Version:          originalVersion,
	}

	if p.version == "" {
		// Will be getting the version from CoreRoller. Need to check for Environment variables
		crApiURL, appID, groupID, err := getContextFromEnvironment(artifact)
		if err != nil {
			return nil, err
		}

		a.AppID = appID
		a.GroupID = groupID

		p.crApiURL = crApiURL
	}

	if err := a.Validate(); err != nil {
		return nil, err
	}

	return a, nil
}

func getLatestPackageFromCoreroller(a *updater.Artifact) (*cr.Package, error) {
	url := fmt.Sprintf("%s/api/apps/%s/groups/%s", p.crApiURL, a.AppID, a.GroupID)

	// setup a transport to handle disgest
	transport := digest.NewTransport(username, password)

	// initialize the client
	client, err := transport.Client()
	if err != nil {
		return nil, err
	}

	// make the call (auth will happen)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	groups := make([]*cr.Group, 0)
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, err
	}

	if len(groups) <= 0 {
		return nil, errors.New("missing group")
	}

	group := groups[0]
	if group.Channel == nil {
		return nil, errors.New("missing channel")
	}

	pkg := group.Channel.Package
	if pkg == nil {
		return nil, errors.New("missing package")
	}

	return pkg, nil
}

func buildUpdateObject(pkg *cr.Package) (*updatr.Update, error) {
	var filename, hash string

	if filenameValue, err := pkg.Filename.Value(); err == nil {
		filename, _ = filenameValue.(string)
	}

	if hashValue, err := pkg.Hash.Value(); err == nil {
		hash, _ = hashValue.(string)
	}

	if filename == "" || pkg.Version == "" || pkg.URL == "" {
		return nil, errors.New("missing some package information")
	}

	return &updatr.Update{
		Version:  pkg.Version,
		URL:      pkg.URL,
		Filename: filename,
		Hash:     hash,
	}, nil
}

func processUpdate(a *updater.Artifact, update *updatr.Update) error {
	logger.Info("downloading artifact", "filename", update.Filename, "URL", update.URL, "version", update.Version)

	artifactPath, err := a.Download(update)
	if err != nil {
		logger.Error("artifact download failed", "filename", update.Filename, "version", update.Version, "error", err)
		return err
	}

	logger.Info("installing update", "version", update.Version)

	if err := a.Install(artifactPath); err != nil {
		logger.Error("install update failed", "artifactPath", artifactPath)
		return err
	}

	logger.Info("update installed", "version", update.Version)
	return nil
}
