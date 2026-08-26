// Copyright 2025 Victor Palma <victor.palma@rackspace.com>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v2

import (
	"reflect"

	"github.com/opencenter-cloud/opencenter-cli/internal/config/services"
)

// applyServiceMapDefaults fills zero-valued BaseConfig fields for services
// present in the config from defaultServiceMap. Enabled is not merged.
func applyServiceMapDefaults(cfg *Config) {
	fqdn := cfg.OpenCenter.Cluster.ClusterFQDN
	if fqdn == "" {
		return
	}
	defs := defaultServiceMap(fqdn)
	mergeServiceMapDefaults(cfg.OpenCenter.Services, defs)
	mergeServiceMapDefaults(cfg.OpenCenter.ManagedServices, defs)
}

func mergeServiceMapDefaults(userServices, defaults ServiceMap) {
	for name, userCfg := range userServices {
		defCfg, ok := defaults[name]
		if !ok {
			continue
		}
		mergeBaseConfig(userCfg, defCfg)
	}
}

func mergeBaseConfig(userCfg, defCfg any) {
	userBase := serviceBaseConfig(userCfg)
	defBase := serviceBaseConfig(defCfg)
	if userBase == nil || defBase == nil {
		return
	}

	if userBase.AdoptionMode == "" {
		userBase.AdoptionMode = defBase.AdoptionMode
	}
	if userBase.Namespace == "" {
		userBase.Namespace = defBase.Namespace
	}
	if userBase.Source.Repo == "" {
		userBase.Source.Repo = defBase.Source.Repo
	}
	if userBase.Source.Branch == "" {
		userBase.Source.Branch = defBase.Source.Branch
	}
	if userBase.Source.Release == "" {
		userBase.Source.Release = defBase.Source.Release
	}
	if userBase.Image.Repository == "" {
		userBase.Image.Repository = defBase.Image.Repository
	}
	if userBase.Image.Tag == "" {
		userBase.Image.Tag = defBase.Image.Tag
	}
	if userBase.Edition == "" {
		userBase.Edition = defBase.Edition
	}
	if userBase.SourceName == "" {
		userBase.SourceName = defBase.SourceName
	}
	if !userBase.SingleStage && defBase.SingleStage {
		userBase.SingleStage = true
	}
	if userBase.HasOverrideValues == nil {
		userBase.HasOverrideValues = defBase.HasOverrideValues
	}
	if !userBase.EnterpriseRegistry && defBase.EnterpriseRegistry {
		userBase.EnterpriseRegistry = true
	}
	if len(userBase.CustomResources) == 0 {
		userBase.CustomResources = defBase.CustomResources
	}
	if len(userBase.ExtraDependencies) == 0 {
		userBase.ExtraDependencies = defBase.ExtraDependencies
	}
	if len(userBase.ConditionalDependencies) == 0 {
		userBase.ConditionalDependencies = defBase.ConditionalDependencies
	}
	if !userBase.BaseOnly && defBase.BaseOnly {
		userBase.BaseOnly = true
	}
	if userBase.KustomizationName == "" {
		userBase.KustomizationName = defBase.KustomizationName
	}
	if len(userBase.OverrideDependsOn) == 0 {
		userBase.OverrideDependsOn = defBase.OverrideDependsOn
	}
	if userBase.OverrideValues == "" {
		userBase.OverrideValues = defBase.OverrideValues
	}
	if userBase.OverrideValuesRendererKey == "" {
		userBase.OverrideValuesRendererKey = defBase.OverrideValuesRendererKey
	}
	if userBase.KustomizationContent == "" {
		userBase.KustomizationContent = defBase.KustomizationContent
	}
	if userBase.OverlayFilesRendererKey == "" {
		userBase.OverlayFilesRendererKey = defBase.OverlayFilesRendererKey
	}
}

func serviceBaseConfig(cfg any) *services.BaseConfig {
	val := reflect.ValueOf(cfg)
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}
	baseField := val.FieldByName("BaseConfig")
	if !baseField.IsValid() || !baseField.CanAddr() {
		return nil
	}
	if base, ok := baseField.Addr().Interface().(*services.BaseConfig); ok {
		return base
	}
	return nil
}

// DefaultServiceConfig returns the default configuration for the named service
// tailored to clusterFQDN, or nil when the service has no registered default.
func DefaultServiceConfig(name, clusterFQDN string) any {
	defs := defaultServiceMap(clusterFQDN)
	cfg, ok := defs[name]
	if !ok {
		return nil
	}
	return cfg
}
