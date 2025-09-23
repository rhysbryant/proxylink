package rulesengine

/*
 Copyright (c) 2025 Rhys Bryant

 This program is free software: you can redistribute it and/or modify
 it under the terms of the GNU General Public License as published by
 the Free Software Foundation, either version 3 of the License, or
 (at your option) any later version.

 This program is distributed in the hope that it will be useful,
 but WITHOUT ANY WARRANTY; without even the implied warranty of
 MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 GNU General Public License for more details.

 You should have received a copy of the GNU General Public License
 along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import (
	"net/url"
	"strings"
)

type RulesEngine struct {
	rules       []Rule
	defaultRule Rule
}

func NewRulesEngine(rules []Rule) *RulesEngine {
	return &RulesEngine{rules: rules}

}

func (re *RulesEngine) GetExitNodes() []ExiteNode {
	exitNodesMap := make(map[string]struct{})
	exitNodes := []ExiteNode{}
	for _, rule := range re.rules {
		if rule.Exit != nil {
			if _, exists := exitNodesMap[rule.Exit.URL]; exists {
				continue
			}
			exitNodesMap[rule.Exit.URL] = struct{}{}
			exitNodes = append(exitNodes, *rule.Exit)
		}
	}
	return exitNodes
}

func hasSuffix(s string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

func (re *RulesEngine) FindMatch(target *url.URL, source string) *Rule {
	targetHost := target.Hostname()
	targetPort := target.Port()
	for _, rule := range re.rules {
		if (len(rule.Target) == 0 || hasSuffix(targetHost, rule.Target)) &&
			(rule.Source == "" || rule.Source == source) &&
			(rule.TargetPort == "" || rule.TargetPort == targetPort) {
			return &rule
		}
	}
	return &re.defaultRule
}
