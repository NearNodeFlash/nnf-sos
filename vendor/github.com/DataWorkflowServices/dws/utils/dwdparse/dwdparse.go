/*
 * Copyright 2021-2024 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dwdparse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DWDirectiveRuleDef defines the DWDirective parser rules
// +kubebuilder:object:generate=true
type DWDirectiveRuleDef struct {
	Key             string   `json:"key"`
	Type            string   `json:"type"`
	Pattern         string   `json:"pattern,omitempty"`
	Patterns        []string `json:"patterns,omitempty"`
	Min             int      `json:"min,omitempty"`
	Max             int      `json:"max,omitempty"`
	IsRequired      bool     `json:"isRequired,omitempty"`
	IsValueRequired bool     `json:"isValueRequired,omitempty"`
	UniqueWithin    string   `json:"uniqueWithin,omitempty"`
}

const emptyValue = "$$empty_value_123456$$"

// DWDirectiveRuleSpec defines the desired state of DWDirective
// +kubebuilder:object:generate=true
type DWDirectiveRuleSpec struct {
	// Name of the #DW command. jobdw, stage_in, etc.
	Command string `json:"command"`

	// Override for the Driver ID. If left empty this defaults to the
	// name of the DWDirectiveRule
	DriverLabel string `json:"driverLabel,omitempty"`

	// Comma separated list of states that this rule wants to register for.
	// These watch states will result in an entry in the driver status array
	// in the Workflow resource
	WatchStates string `json:"watchStates,omitempty"`

	// List of key/value pairs this #DW command is expected to have
	RuleDefs []DWDirectiveRuleDef `json:"ruleDefs"`
}

// BuildArgsMap builds a map of the DWDirective's arguments in the form: args["key"] = value
func BuildArgsMap(dwd string) (map[string]string, error) {

	dwdArgs := strings.Fields(dwd)

	if len(dwdArgs) == 0 || dwdArgs[0] != "#DW" {
		return nil, fmt.Errorf("missing '#DW' prefix in directive '%s'", dwd)
	}

	argsMap := make(map[string]string)
	argsMap["command"] = dwdArgs[1]
	for i := 2; i < len(dwdArgs); i++ {
		keyValue := strings.Split(dwdArgs[i], "=")

		// Don't allow repeated arguments
		_, ok := argsMap[keyValue[0]]
		if ok {
			return nil, fmt.Errorf("repeated argument '%s' in directive '%s'", keyValue[0], dwd)
		}

		if len(keyValue) == 1 {
			// We won't know how to interpret an empty value--whether it's allowed,
			// or what its type should be--until ValidateArgs().
			argsMap[keyValue[0]] = emptyValue
		} else if len(keyValue) == 2 {
			argsMap[keyValue[0]] = keyValue[1]
		} else {
			keyValue := strings.SplitN(dwdArgs[i], "=", 2)
			argsMap[keyValue[0]] = keyValue[1]
		}
	}

	return argsMap, nil
}

// Compile this regex outside the loop for better performance.
var boolMatcher = regexp.MustCompile(`(?i)^(true|false)$`) // (?i) -> case-insensitve comparison

// ValidateArgs validates a map of arguments against the rule specification
func ValidateArgs(spec DWDirectiveRuleSpec, args map[string]string, uniqueMap map[string]bool) error {

	command, found := args["command"]
	if !found {
		return fmt.Errorf("no command in arguments")
	}

	if command != spec.Command {
		return fmt.Errorf("command '%s' does not match rule '%s'", command, spec.Command)
	}

	// Create a map that maps a directive rule definition to an argument that correctly matches it
	// key: DWDirectiveRule	value: argument that matches that rule
	// Required to check that all DWDirectiveRuleDef's have been met
	argToRuleMap := map[*DWDirectiveRuleDef]string{}

	findRuleDefinition := func(key string) (*DWDirectiveRuleDef, error) {
		for index, rule := range spec.RuleDefs {
			re, err := regexp.Compile(rule.Key)
			if err != nil {
				return nil, fmt.Errorf("invalid rule regular expression '%s'", rule.Key)
			}

			if re.MatchString(key) {
				return &spec.RuleDefs[index], nil
			}
		}

		return nil, fmt.Errorf("unsupported argument '%s'", key)
	}

	// Iterate over all arguments and validate each based on the associated rule
	for k, v := range args {
		if k == "command" {
			continue
		}

		rule, err := findRuleDefinition(k)
		if err != nil {
			return err
		}

		if v == emptyValue {
			if rule.Type == "bool" && !rule.IsValueRequired {
				// Booleans default to true.
				v = "true"
			} else if rule.IsValueRequired {
				return fmt.Errorf("argument '%s' requires value", k)
			}
		}

		switch rule.Type {
		case "integer":
			i, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("argument '%s' invalid integer '%s'", k, v)
			}
			if rule.Max != 0 && i > rule.Max {
				return fmt.Errorf("argument '%s' specified integer value %d is greather than maximum value %d", k, i, rule.Max)
			}
			if rule.Min != 0 && i < rule.Min {
				return fmt.Errorf("argument '%s' specified integer value %d is less than minimum value %d", k, i, rule.Min)
			}
		case "bool":
			if rule.Pattern != "" {
				if !boolMatcher.MatchString(v) {
					return fmt.Errorf("argument '%s' invalid boolean '%s'", k, v)
				}
			}
		case "string":
			if rule.Pattern != "" {
				re, err := regexp.Compile(rule.Pattern)
				if err != nil {
					return fmt.Errorf("invalid regular expression '%s'", rule.Pattern)
				}

				if !re.MatchString(v) {
					return fmt.Errorf("argument '%s' invalid string '%s'", k, v)
				}
			}
		case "list-of-string":
			words := strings.Split(v, ",")
			if len(rule.Patterns) > 0 {
				wordsMatched := make(map[string]bool)
				for _, word := range words {
					for idx := range rule.Patterns {
						re, err := regexp.Compile(rule.Patterns[idx])
						if err != nil {
							return fmt.Errorf("invalid regular expression '%s'", rule.Patterns[idx])
						}

						if re.MatchString(word) {
							wordsMatched[word] = true
							break
						}
					}
					if !wordsMatched[word] {
						return fmt.Errorf("argument '%s' invalid string '%s' in list '%s'", k, word, v)
					}
				}
			}
			// All of the words in the list are valid, but were any words repeated?
			wordMap := make(map[string]bool)
			for _, word := range words {
				if _, present := wordMap[word]; present {
					return fmt.Errorf("argument '%s' word '%s' repeated in list '%s'", k, word, v)
				}
				wordMap[word] = true
			}
		default:
			return fmt.Errorf("unsupported rule type '%s'", rule.Type)
		}

		if rule.UniqueWithin != "" {
			_, ok := uniqueMap[rule.UniqueWithin+"/"+v]
			if ok {
				return fmt.Errorf("value '%s' must be unique within '%s'", v, rule.UniqueWithin)
			}

			uniqueMap[rule.UniqueWithin+"/"+v] = true
		}

		// NOTE: We know that we don't have repeated arguments here because the arguments
		//       come to us in a map indexed by the argment name.
		argToRuleMap[rule] = k
	}

	// Iterate over the rules to ensure all required rules have an argument
	for index := range spec.RuleDefs {
		rule := &spec.RuleDefs[index]
		if rule.IsRequired {
			if _, found := argToRuleMap[rule]; !found {
				return fmt.Errorf("missing required argument '%v'", rule.Key)
			}
		}
	}

	return nil
}

// Validate a list of directives against the supplied rules. When a directive is valid
// for a particular rule, the `onValidDirectiveFunc` function is called.
func Validate(rules []DWDirectiveRuleSpec, directives []string, onValidDirectiveFunc func(index int, rule DWDirectiveRuleSpec)) error {

	// Create a map to track argument uniqueness within the directives for
	// rules that contain `UniqueWithin`
	uniqueMap := make(map[string]bool)

	for index, directive := range directives {

		// A directive is validated against all rules; any one rule that is valid for a directive
		// makes that directive valid.
		validDirective := false

		for _, rule := range rules {
			valid, err := validateDWDirective(rule, directive, uniqueMap)
			if err != nil {
				return err
			}

			if valid {
				validDirective = true
				onValidDirectiveFunc(index, rule)
			}
		}

		if !validDirective {
			return fmt.Errorf("invalid directive '%s'", directive)
		}
	}

	return nil
}

// validateDWDirective validates an individual directive against a rule
func validateDWDirective(rule DWDirectiveRuleSpec, dwd string, uniqueMap map[string]bool) (bool, error) {

	// Build a map of the #DW commands and arguments
	argsMap, err := BuildArgsMap(dwd)
	if err != nil {
		return false, err
	}

	if argsMap["command"] != rule.Command {
		return false, nil
	}

	err = ValidateArgs(rule, argsMap, uniqueMap)
	if err != nil {
		return false, err
	}

	return true, nil
}
