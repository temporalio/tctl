// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package config

import (
	"errors"
	"strings"

	"gopkg.in/yaml.v3"
)

func (cfg *Config) SetAliasValue(command string, value string) error {
	aliasesRoot, err := cfg.getRecord("aliases")
	if err != nil {
		return err
	}

	if aliasesRoot.Kind != yaml.SequenceNode {
		// empty node is read as scalar node
		aliasesRoot.Kind = yaml.SequenceNode
		aliasesRoot.Tag = "!!seq"
	}

	alias, err := cfg.getAliasRecord(command)
	if err != nil {
		alias = createAliasRecord(command, value)
		aliasesRoot.Content = append(aliasesRoot.Content, alias)
	} else {
		alias.Content[3].Value = value // set command alias value
	}

	return writeConfig(cfg)
}

func addAliasesRoot(cfg *Config) error {
	_, err := cfg.getRecord("aliases")
	if err != nil {
		aliasKey := &yaml.Node{
			Value: "aliases",
			Kind:  yaml.ScalarNode,
		}
		aliasSeq := &yaml.Node{
			Kind: yaml.SequenceNode,
		}
		cfg.Root.Content[0].Content = append(cfg.Root.Content[0].Content, aliasKey, aliasSeq)
	}
	return nil
}

func (cfg *Config) getAliasRecord(command string) (*yaml.Node, error) {
	aliases, err := cfg.getRecord("aliases")
	if err != nil {
		return nil, err
	}

	for _, aliasContainer := range aliases.Content {
		keyName := aliasContainer.Content[1].Value

		if strings.Compare(command, keyName) == 0 {
			return aliasContainer, nil
		}
	}

	return nil, errors.New("unable to find an alias for command " + command)
}

func createAliasRecord(command string, value string) *yaml.Node {
	commandNameKey := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "command",
	}
	commandName := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: command,
	}

	commandValueKey := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "value",
	}

	commandValue := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: value,
	}

	container := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			commandNameKey,
			commandName,
			commandValueKey,
			commandValue,
		},
	}

	return container
}
