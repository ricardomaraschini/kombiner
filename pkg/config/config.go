/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"bytes"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	configapi "kombiner/pkg/apis/config/v1alpha1"
)

// fromFile provides an alternative to the deprecated ctrl.ConfigFile().AtPath(path).OfKind(&cfg)
func fromFile(path string, scheme *runtime.Scheme, cfg *configapi.Configuration) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	codecs := serializer.NewCodecFactory(scheme, serializer.EnableStrict)

	// Regardless of if the bytes are of any external version,
	// it will be read successfully and converted into the internal version
	return runtime.DecodeInto(codecs.UniversalDecoder(), content, cfg)
}

func Encode(scheme *runtime.Scheme, cfg *configapi.Configuration) (string, error) {
	codecs := serializer.NewCodecFactory(scheme)
	const mediaType = runtime.ContentTypeYAML
	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return "", fmt.Errorf("unable to locate encoder -- %q is not a supported media type", mediaType)
	}

	encoder := codecs.EncoderForVersion(info.Serializer, configapi.GroupVersion)
	buf := new(bytes.Buffer)
	if err := encoder.Encode(cfg, buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Load returns a set of controller options and configuration from the given file, if the config file path is empty
// it used the default configapi values.
func Load(scheme *runtime.Scheme, configFile string) (configapi.Configuration, error) {
	cfg := configapi.Configuration{}
	err := fromFile(configFile, scheme, &cfg)
	if err != nil {
		return cfg, err
	}

	if err := validate(&cfg).ToAggregate(); err != nil {
		return cfg, err
	}

	return cfg, err
}
