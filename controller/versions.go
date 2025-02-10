package controller

import (
	"encoding/json"
	"fmt"
	corootv1 "github.io/coroot/operator/api/v1"
	"io"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

type App string

const (
	AppCorootCE     App = "coroot"
	AppCorootEE     App = "coroot-ee"
	AppNodeAgent    App = "coroot-node-agent"
	AppClusterAgent App = "coroot-cluster-agent"
)

func (r *CorootReconciler) getAppImage(cr *corootv1.Coroot, app App) string {
	var v string
	switch app {
	case AppCorootCE:
		v = cr.Spec.CommunityEdition.Version
	case AppCorootEE:
		v = cr.Spec.EnterpriseEdition.Version
	case AppNodeAgent:
		v = cr.Spec.NodeAgent.Version
	case AppClusterAgent:
		v = cr.Spec.ClusterAgent.Version
	}
	if v == "" {
		r.versionsLock.Lock()
		defer r.versionsLock.Unlock()
		v = r.versions[app]
		if v == "" {
			v = "latest"
		}
	}
	if strings.Contains(v, ":") {
		return v
	}
	return fmt.Sprintf("ghcr.io/coroot/%s:%s", app, v)
}

func (r *CorootReconciler) fetchAppVersions() {
	logger := log.FromContext(nil)
	versions := map[App]string{}
	for _, app := range []App{AppCorootCE, AppCorootEE, AppNodeAgent, AppClusterAgent} {
		v, err := r.fetchAppVersion(app)
		if err != nil {
			logger.Error(err, "failed to get version", "app", app)
		}
		versions[app] = v
	}
	logger.Info(fmt.Sprintf("got app versions: %v", versions))
	r.versionsLock.Lock()
	defer r.versionsLock.Unlock()
	for app, v := range versions {
		if v != "" {
			r.versions[app] = v
		}
	}
}

func (r *CorootReconciler) fetchAppVersion(app App) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/coroot/%s/releases/latest", app))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err = json.Unmarshal(data, &release); err != nil {
		return "", err
	}
	return strings.TrimPrefix(release.TagName, "v"), nil
}
