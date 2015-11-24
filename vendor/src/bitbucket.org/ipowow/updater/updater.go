package updater

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/bugsnag/osext"
	"github.com/coreos/go-semver/semver"
	cr "github.com/coreroller/coreroller/updaters/lib/go"
	"github.com/mgutz/logxi/v1"
)

const (
	verSep = "_v"
)

var (
	logger = log.New("updater")

	// ErrMissingInstanceID is the error returned when CR_INSTANCE_ID is not provided
	ErrMissingInstanceID = errors.New("CR_INSTANCE_ID environment variable is missing")

	// ErrMissingOmahaURL is the error returned when CR_OMAHA_URL is not provided
	ErrMissingOmahaURL = errors.New("CR_OMAHA_URL environment variable is missing")

	// ErrMissingAppID is the error returned when xxx_CR_APP_ID is not provided
	ErrMissingAppID = errors.New("<artifact>_CR_APP_ID environment variable is missing")

	// ErrMissingGroupID is the error returned when xxx_CR_GROUP_ID is not provided
	ErrMissingGroupID = errors.New("<artifact>_CR_GROUP_ID environment variable is missing")

	// ErrIncorrectFilenameFormat is the error returned when the artifact filename is not in the right format
	ErrIncorrectFilenameFormat = errors.New("filename does not follow updater naming convention (executable_v1.0.2)")
)

type Updater struct {
	checkFrequency time.Duration
	signal         syscall.Signal
	a              *Artifact
}

// New instantiates the updater instance so it checks for update every `checkFrequency`.
func New(checkFrequency time.Duration, signal syscall.Signal) (*Updater, error) {
	artifact, err := getArtifactContextFromEnvironment()
	if err != nil {
		logger.Error("NewUpdater", "error", err.Error())
		return nil, err
	}

	u := &Updater{
		checkFrequency: checkFrequency,
		signal:         signal,
		a:              artifact,
	}

	return u, nil
}

func getArtifactContextFromEnvironment() (*Artifact, error) {
	executable, err := osext.Executable()
	if err != nil {
		return nil, err
	}

	dir, _ := filepath.Split(executable)
	prefix, version, err := getPrefixAndVersion(executable)
	if err != nil {
		return nil, err
	}

	instanceID, omahaURL, appID, groupID, err := getContextFromEnvironment(prefix)
	if err != nil {
		return nil, err
	}

	a := &Artifact{
		ExecutableDir:    dir,
		ExecutablePrefix: prefix,
		Version:          version,
		OmahaURL:         omahaURL,
		InstanceID:       instanceID,
		AppID:            appID,
		GroupID:          groupID,
	}

	if err := a.Validate(); err != nil {
		return nil, err
	}

	return a, nil
}

func getContextFromEnvironment(prefix string) (instanceID string, omahaURL string, appID string, groupID string, err error) {
	instanceID = os.Getenv("CR_INSTANCE_ID")
	if instanceID == "" {
		err = ErrMissingInstanceID
		return
	}

	omahaURL = os.Getenv("CR_OMAHA_URL")
	if omahaURL == "" {
		err = ErrMissingOmahaURL
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

func getPrefixAndVersion(executable string) (string, string, error) {
	executable, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", "", err
	}
	_, file := filepath.Split(executable)

	fileParts := strings.Split(file, verSep)
	if len(fileParts) != 2 {
		return "", "", ErrIncorrectFilenameFormat
	}

	prefix := fileParts[0]
	version, err := semver.NewVersion(fileParts[1])
	if err != nil {
		return "", "", err
	}

	return prefix, version.String(), nil
}

func (u *Updater) Start() {
	t := time.Tick(u.checkFrequency)

	for range t {
		update, err := cr.GetUpdate(u.a.InstanceID, u.a.AppID, u.a.GroupID, u.a.Version)
		switch err {
		case nil:
		case cr.ErrNoUpdate:
			logger.Debug("no update from CoreRoller")
			continue
		default:
			logger.Warn("Getting update from CoreRoller", "appID", u.a.AppID, "groupID", u.a.GroupID, "instanceID", u.a.InstanceID, "error", err.Error())
			continue
		}

		logger.Info("got update from coreroller", "version", update.Version)
		if err := u.processUpdate(update); err != nil {
			cr.EventUpdateFailed(u.a.InstanceID, u.a.AppID, u.a.GroupID)
			continue
		}

		break
	}
}

func (u *Updater) processUpdate(update *cr.Update) error {
	logger.Info("downloading artifact", "filename", update.Filename, "version", update.Version)
	cr.EventDownloadStarted(u.a.InstanceID, u.a.AppID, u.a.GroupID)

	artifactPath, err := u.a.Download(update)
	if err != nil {
		logger.Error("artifact download failed", "filename", update.Filename, "version", update.Version, "error", err)
		return err
	}

	logger.Info("artifact downloaded", "artifactPath", artifactPath)
	cr.EventDownloadFinished(u.a.InstanceID, u.a.AppID, u.a.GroupID)

	logger.Info("installing update", "version", update.Version)

	if err := u.a.Install(artifactPath); err != nil {
		logger.Error("install update failed", "artifactPath", artifactPath)
		return err
	}

	logger.Info("update installed", "version", update.Version)
	cr.EventUpdateSucceeded(u.a.InstanceID, u.a.AppID, u.a.GroupID)

	logger.Info("sending signal to process now to restart..", "version", update.Version)
	syscall.Kill(syscall.Getpid(), u.signal)

	return nil
}
