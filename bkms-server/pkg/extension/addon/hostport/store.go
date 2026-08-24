/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package hostport

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const collectionName = "hostport_configs"

var (
	// ErrConfigNotFound is returned when the app has no HostPort config document.
	ErrConfigNotFound = errors.New("hostport config not found")
	// ErrPortExists is returned when adding a duplicate container port.
	ErrPortExists = errors.New("hostport container port already exists")
	// ErrPortNotFound is returned when deleting a port that is not declared.
	ErrPortNotFound = errors.New("hostport container port not found")
	// ErrInvalidPort is returned when the container port is out of range.
	ErrInvalidPort = errors.New("hostport container port is invalid")
)

// HostPortStore persists HostPortConfig documents.
type HostPortStore interface {
	Get(ctx context.Context, appID string) (*HostPortConfig, error)
	Ensure(ctx context.Context, appID string) (*HostPortConfig, error)
	ListPorts(ctx context.Context, appID string) ([]int32, error)
	AddPort(ctx context.Context, appID string, port int32) (*HostPortConfig, error)
	RemovePort(ctx context.Context, appID string, port int32) error
	UpsertEnvState(ctx context.Context, appID, envName string, appliedPorts []int32) error
	RemoveEnvState(ctx context.Context, appID, envName string) error
	DeleteByApp(ctx context.Context, appID string) error
}

var _ HostPortStore = &HostPortStoreMongo{}

// HostPortStoreMongo is the MongoDB implementation of HostPortStore.
type HostPortStoreMongo struct {
	collection *mongo.Collection
}

// NewHostPortStoreMongo creates a HostPortStore backed by MongoDB.
func NewHostPortStoreMongo(client *mongo.Client, dbName string) (HostPortStore, error) {
	coll := client.Database(dbName).Collection(collectionName)
	// Index (managed by golang-migrate): unique appID
	return &HostPortStoreMongo{collection: coll}, nil
}

// Get returns the HostPort config for an app.
func (s *HostPortStoreMongo) Get(ctx context.Context, appID string) (*HostPortConfig, error) {
	var config HostPortConfig
	err := s.collection.FindOne(ctx, bson.M{"appID": appID}).Decode(&config)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrConfigNotFound
		}
		return nil, errors.Wrap(err, "find hostport config")
	}
	if config.Ports == nil {
		config.Ports = []int32{}
	}
	if config.EnvStates == nil {
		config.EnvStates = map[string]HostPortEnvState{}
	}
	return &config, nil
}

// Ensure returns the existing config or creates an empty one.
func (s *HostPortStoreMongo) Ensure(ctx context.Context, appID string) (*HostPortConfig, error) {
	config, err := s.Get(ctx, appID)
	if err == nil {
		return config, nil
	}
	if !errors.Is(err, ErrConfigNotFound) {
		return nil, err
	}

	now := time.Now()
	config = &HostPortConfig{
		AppID:     appID,
		Ports:     []int32{},
		EnvStates: map[string]HostPortEnvState{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err = s.collection.InsertOne(ctx, config)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return s.Get(ctx, appID)
		}
		return nil, errors.Wrap(err, "insert hostport config")
	}
	return config, nil
}

// ListPorts returns declared container ports; missing config yields an empty slice.
func (s *HostPortStoreMongo) ListPorts(ctx context.Context, appID string) ([]int32, error) {
	config, err := s.Get(ctx, appID)
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return []int32{}, nil
		}
		return nil, err
	}
	return NormalizePorts(config.Ports), nil
}

// AddPort appends a container port to the app mapping (lazy-creates the document).
func (s *HostPortStoreMongo) AddPort(ctx context.Context, appID string, port int32) (*HostPortConfig, error) {
	if !ValidateContainerPort(port) {
		return nil, ErrInvalidPort
	}

	config, err := s.Ensure(ctx, appID)
	if err != nil {
		return nil, err
	}
	for _, existing := range config.Ports {
		if existing == port {
			return nil, ErrPortExists
		}
	}

	ports := NormalizePorts(append(append([]int32{}, config.Ports...), port))
	now := time.Now()
	result, err := s.collection.UpdateOne(
		ctx,
		bson.M{"appID": appID},
		bson.M{"$set": bson.M{"ports": ports, "updatedAt": now}},
	)
	if err != nil {
		return nil, errors.Wrap(err, "add hostport mapping")
	}
	if result.MatchedCount == 0 {
		return nil, ErrConfigNotFound
	}
	config.Ports = ports
	config.UpdatedAt = now
	return config, nil
}

// RemovePort deletes a container port from the app mapping.
func (s *HostPortStoreMongo) RemovePort(ctx context.Context, appID string, port int32) error {
	if !ValidateContainerPort(port) {
		return ErrInvalidPort
	}

	config, err := s.Get(ctx, appID)
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return ErrPortNotFound
		}
		return err
	}

	found := false
	remaining := make([]int32, 0, len(config.Ports))
	for _, existing := range config.Ports {
		if existing == port {
			found = true
			continue
		}
		remaining = append(remaining, existing)
	}
	if !found {
		return ErrPortNotFound
	}

	remaining = NormalizePorts(remaining)
	result, err := s.collection.UpdateOne(
		ctx,
		bson.M{"appID": appID},
		bson.M{"$set": bson.M{"ports": remaining, "updatedAt": time.Now()}},
	)
	if err != nil {
		return errors.Wrap(err, "remove hostport mapping")
	}
	if result.MatchedCount == 0 {
		return ErrConfigNotFound
	}
	return nil
}

// UpsertEnvState records applied ports for an environment (ensures document exists).
func (s *HostPortStoreMongo) UpsertEnvState(
	ctx context.Context,
	appID, envName string,
	appliedPorts []int32,
) error {
	if _, err := s.Ensure(ctx, appID); err != nil {
		return err
	}
	fieldPrefix, err := envFieldPrefix("envStates", envName)
	if err != nil {
		return err
	}
	appliedPorts = NormalizePorts(appliedPorts)
	setFields := bson.M{
		fieldPrefix + ".appliedPorts": appliedPorts,
		fieldPrefix + ".updatedAt":    time.Now(),
		"updatedAt":                   time.Now(),
	}
	result, err := s.collection.UpdateOne(ctx, bson.M{"appID": appID}, bson.M{"$set": setFields})
	if err != nil {
		return errors.Wrap(err, "upsert hostport env state")
	}
	if result.MatchedCount == 0 {
		return ErrConfigNotFound
	}
	return nil
}

// RemoveEnvState removes the env snapshot after uninstall.
func (s *HostPortStoreMongo) RemoveEnvState(ctx context.Context, appID, envName string) error {
	fieldPrefix, err := envFieldPrefix("envStates", envName)
	if err != nil {
		return err
	}
	result, err := s.collection.UpdateOne(
		ctx,
		bson.M{"appID": appID},
		bson.M{
			"$unset": bson.M{fieldPrefix: ""},
			"$set":   bson.M{"updatedAt": time.Now()},
		},
	)
	if err != nil {
		return errors.Wrap(err, "remove hostport env state")
	}
	if result.MatchedCount == 0 {
		// No config yet — nothing to clean.
		return nil
	}
	return nil
}

// DeleteByApp deletes the HostPort config for an app (tests / cleanup).
func (s *HostPortStoreMongo) DeleteByApp(ctx context.Context, appID string) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"appID": appID})
	if err != nil {
		return errors.Wrap(err, "delete hostport config by app")
	}
	return nil
}

func envFieldPrefix(root, envName string) (string, error) {
	if envName == "" || strings.ContainsAny(envName, ".$") {
		return "", errors.Errorf("invalid env name %q", envName)
	}
	return root + "." + envName, nil
}
