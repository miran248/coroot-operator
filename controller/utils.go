package controller

import (
	"crypto/rand"
	"encoding/json"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"math/big"
	"reflect"
)

func GeneratePassword(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	res := make([]byte, length)
	for i := range res {
		randomIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		res[i] = charset[randomIndex.Int64()]
	}
	return string(res)
}

// Merge merges `overrides` into `base` using the SMP (structural merge patch) approach.
// - It intentionally does not remove fields present in base but missing from overrides
// - It merges slices only if the `patchStrategy:"merge"` tag is present and the `patchMergeKey` identifies the unique field
func Merge[T any](base *T, overrides T) error {
	baseBytes, err := json.Marshal(base)
	if err != nil {
		return errors.Wrap(err, "failed to convert current object to byte sequence")
	}

	overrideBytes, err := json.Marshal(overrides)
	if err != nil {
		return errors.Wrap(err, "failed to convert current object to byte sequence")
	}

	patchMeta, err := strategicpatch.NewPatchMetaFromStruct(base)
	if err != nil {
		return errors.Wrap(err, "failed to produce patch meta from struct")
	}
	patch, err := strategicpatch.CreateThreeWayMergePatch(overrideBytes, overrideBytes, baseBytes, patchMeta, true)
	if err != nil {
		return errors.Wrap(err, "failed to create three way merge patch")
	}

	merged, err := strategicpatch.StrategicMergePatchUsingLookupPatchMeta(baseBytes, patch, patchMeta)
	if err != nil {
		return errors.Wrap(err, "failed to apply patch")
	}

	valueOfBase := reflect.Indirect(reflect.ValueOf(base))
	into := reflect.New(valueOfBase.Type())
	if err := json.Unmarshal(merged, into.Interface()); err != nil {
		return err
	}
	if !valueOfBase.CanSet() {
		return errors.New("unable to set unmarshalled value into base object")
	}
	valueOfBase.Set(reflect.Indirect(into))
	return nil
}
