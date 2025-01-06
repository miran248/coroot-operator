package controller

import (
	"fmt"
	corootv1 "github.io/coroot/operator/api/v1"
	"sort"
	"strings"
)

func postgresConnectionString(p corootv1.PostgresSpec, passwordEnvVar string) string {
	kv := p.Params
	if kv == nil {
		kv = map[string]string{}
	}
	kv["host"] = p.Host
	if p.Port > 0 {
		kv["port"] = fmt.Sprintf("%d", p.Port)
	}
	kv["user"] = p.User
	kv["password"] = fmt.Sprintf("$(%s)", passwordEnvVar)
	kv["dbname"] = p.Database
	if kv["sslmode"] == "" {
		kv["sslmode"] = "disable"
	}
	var kvs []string
	for k, v := range kv {
		if v != "" {
			kvs = append(kvs, fmt.Sprintf("%s='%s'", k, v))
		}
	}
	sort.Strings(kvs)
	return strings.Join(kvs, " ")
}
