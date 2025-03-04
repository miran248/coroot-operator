package controller

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AnnotationLastAppliedConfiguration = "operator.coroot.com/last-applied-configuration"
	RandomStringCharset                = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func RandomString(length int) string {
	res := make([]byte, length)
	for i := range res {
		randomIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(RandomStringCharset))))
		res[i] = RandomStringCharset[randomIndex.Int64()]
	}
	return string(res)
}

type LastAppliedConfiguration struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Spec        json.RawMessage   `json:"spec,omitempty"`
}

func MergeSpecs[T any](obj client.Object, currentSpec *T, targetSpec T, targetAnnotations map[string]string) error {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	current, err := json.Marshal(currentSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal current: %w", err)
	}
	target, err := json.Marshal(targetSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal target: %w", err)
	}

	var original []byte
	lastApplied := []byte(annotations[AnnotationLastAppliedConfiguration])
	var cfg LastAppliedConfiguration
	if err = json.Unmarshal(lastApplied, &cfg); err != nil {
		original = lastApplied
	} else {
		original = cfg.Spec
		for k := range cfg.Annotations {
			delete(annotations, k)
		}
	}
	for k, v := range targetAnnotations {
		annotations[k] = v
	}
	cfg.Annotations = targetAnnotations
	cfg.Spec = target
	lastApplied, err = json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal last applied: %w", err)
	}
	annotations[AnnotationLastAppliedConfiguration] = string(lastApplied)
	obj.SetAnnotations(annotations)

	patchMeta, err := strategicpatch.NewPatchMetaFromStruct(currentSpec)
	if err != nil {
		return fmt.Errorf("failed to produce patch meta from struct: %w", err)
	}
	patch, err := strategicpatch.CreateThreeWayMergePatch(original, target, current, patchMeta, true)
	if err != nil {
		return fmt.Errorf("failed to create three way merge patch: %w", err)
	}

	if string(patch) == "{}" {
		return nil
	}

	merged, err := strategicpatch.StrategicMergePatchUsingLookupPatchMeta(current, patch, patchMeta)
	if err != nil {
		return fmt.Errorf("failed to apply patch: %w", err)
	}

	valueOfCurrent := reflect.Indirect(reflect.ValueOf(currentSpec))
	into := reflect.New(valueOfCurrent.Type())
	if err = json.Unmarshal(merged, into.Interface()); err != nil {
		return fmt.Errorf("failed to unmarshal merged data: %w", err)
	}
	if !valueOfCurrent.CanSet() {
		return fmt.Errorf("unable to set unmarshalled value into current object")
	}
	valueOfCurrent.Set(reflect.Indirect(into))
	return nil
}

func secretKeySelector(name, key string) *corev1.SecretKeySelector {
	return &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: name,
		},
		Key: key,
	}
}
