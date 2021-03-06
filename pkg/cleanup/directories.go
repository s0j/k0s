package cleanup

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/mount-utils"
)

type directories struct {
	Config *Config
}

// Name returns the name of the step
func (d *directories) Name() string {
	return "remove directories step"
}

// NeedsToRun checks if dataDir and runDir are present on the host
func (d *directories) NeedsToRun() bool {
	if _, err := os.Stat(d.Config.dataDir); err == nil {
		return true
	}
	if _, err := os.Stat(d.Config.runDir); err == nil {
		return true
	}
	return false
}

// Run removes all kubelet mounts and deletes generated dataDir and runDir
func (d *directories) Run() error {
	// unmount any leftover overlays (such as in alpine)
	mounter := mount.New("")
	procMounts, err := mounter.List()
	if err != nil {
		return err
	}

	// search and unmount kubelet volume mounts
	for _, v := range procMounts {
		if strings.Compare(v.Path, fmt.Sprintf("%s/kubelet", d.Config.dataDir)) == 0 {
			logrus.Debugf("%v is mounted! attempting to unmount...", v.Path)
			if err = mounter.Unmount(v.Path); err != nil {
				logrus.Warningf("failed to unmount %v", v.Path)
			}
		} else if strings.Compare(v.Path, d.Config.dataDir) == 0 {
			logrus.Debugf("%v is mounted! attempting to unmount...", v.Path)
			if err = mounter.Unmount(v.Path); err != nil {
				logrus.Warningf("failed to unmount %v", v.Path)
			}
		}
	}

	logrus.Debugf("deleting k0s generated data-dir (%v) and run-dir (%v)", d.Config.dataDir, d.Config.runDir)
	if err := os.RemoveAll(d.Config.dataDir); err != nil {
		fmtError := fmt.Errorf("failed to delete %v. err: %v", d.Config.dataDir, err)
		return fmtError
	}
	if err := os.RemoveAll(d.Config.runDir); err != nil {
		fmtError := fmt.Errorf("failed to delete %v. err: %v", d.Config.runDir, err)
		return fmtError
	}

	return nil
}
