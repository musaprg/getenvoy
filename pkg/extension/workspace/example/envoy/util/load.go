// Copyright 2020 Tetrate
//
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

package util

import (
	"fmt"

	envoymatcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/runtime/protoimpl"
	"sigs.k8s.io/yaml"

	"github.com/tetratelabs/getenvoy/pkg/extension/workspace/model"
)

// Load loads Envoy configuration from a YAML or JSON file.
func Load(config *model.File, message proto.Message) error {
	json, err := yaml.YAMLToJSON(config.Content)
	if err != nil {
		return fmt.Errorf("failed to convert into JSON Envoy config coming from %q: %w", config.Source, err)
	}

	unmarshaller := protojson.UnmarshalOptions{
		DiscardUnknown: true,
		// ignore unknown extension types that might appear in a user-defined Envoy config
		Resolver: fakeAnyResolver{},
	}
	return unmarshaller.Unmarshal(json, message)
}

// fakeAnyResolver resolves any type URL into a wrong but known message type.
//
// When using fakeAnyResolver, the actual payload of a google.protobuf.Any field
// will be lost during unmarshalling.
// On the bright side, unmarshalling will never fail due to use of unknown types.
type fakeAnyResolver struct{}

func (r fakeAnyResolver) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	return r.FindMessageByURL(string(message))
}

func (r fakeAnyResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	// NOTE: as of github.com/golang/protobuf/jsonpb@v1.3.x, the only way to achieve
	// desired behavior is to return a Protobuf message that has no fields.
	// As an extra constraint, it cannot be a well-known type, such as google.protobuf.Empty.
	// That is why we are reusing an empty message type that already exists in Envoy API.
	m := new(envoymatcher.ValueMatcher_NullMatch)
	return protoimpl.X.MessageTypeOf(m), nil
}

func (r fakeAnyResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return protoregistry.GlobalTypes.FindExtensionByName(field)
}

func (r fakeAnyResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return protoregistry.GlobalTypes.FindExtensionByNumber(message, field)
}
