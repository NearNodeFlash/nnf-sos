/*
 * Copyright 2021, 2022 Hewlett Packard Enterprise Development LP
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
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DWDirectiveRuleDef defines the DWDirective parser rules
// +kubebuilder:object:generate=true
type DWDirectiveRuleDef struct {
	Key             string `json:"key"`
	Type            string `json:"type"`
	Pattern         string `json:"pattern,omitempty"`
	Min             int    `json:"min,omitempty"`
	Max             int    `json:"max,omitempty"`
	IsRequired      bool   `json:"isRequired,omitempty"`
	IsValueRequired bool   `json:"isValueRequired,omitempty"`
	UniqueWithin    string `json:"uniqueWithin,omitempty"`
}

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

type dwUnsupportedCommandErr struct {
	command string
}

// NewUnsupportedCommandErr returns a reference to the unsupported command type
func NewUnsupportedCommandErr(command string) error {
	return &dwUnsupportedCommandErr{command}
}

func (e *dwUnsupportedCommandErr) Error() string {
	return fmt.Sprintf("Unsupported Command: '%s'", e.command)
}

// IsUnsupportedCommand returns true if the error indicates that the command
// is unsupported
func IsUnsupportedCommand(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*dwUnsupportedCommandErr)
	return ok
}

// BuildRulesMap builds a map of the DWDirectives argument parser rules for the specified command
func BuildRulesMap(rule DWDirectiveRuleSpec, cmd string) (map[string]DWDirectiveRuleDef, error) {
	rulesMap := make(map[string]DWDirectiveRuleDef)

	for _, rd := range rule.RuleDefs {
		rulesMap[rd.Key] = rd
	}

	if len(rulesMap) == 0 {
		return nil, NewUnsupportedCommandErr(cmd)
	}

	return rulesMap, nil
}

// BuildArgsMap builds a map of the DWDirective's arguments in the form: args["key"] = value
func BuildArgsMap(dwd string) (map[string]string, error) {
	argsMap := make(map[string]string)
	dwdArgs := strings.Fields(dwd)

	if len(dwdArgs) == 0 {
		return nil, fmt.Errorf("Invalid format for directive '%s'", dwd)
	}

	if dwdArgs[0] == "#DW" {
		argsMap["command"] = dwdArgs[1]
		for i := 2; i < len(dwdArgs); i++ {
			keyValue := strings.Split(dwdArgs[i], "=")

			// Don't allow repeated arguments
			_, ok := argsMap[keyValue[0]]
			if ok {
				return nil, errors.New("repeated argument in directive: " + keyValue[0])
			}

			if len(keyValue) == 1 {
				argsMap[keyValue[0]] = "true"
			} else if len(keyValue) == 2 {
				argsMap[keyValue[0]] = keyValue[1]
			} else {
				keyValue := strings.SplitN(dwdArgs[i], "=", 2)
				argsMap[keyValue[0]] = keyValue[1]
			}
		}
	} else {
		return nil, errors.New("missing #DW in directive")
	}
	return argsMap, nil
}

// ValidateArgs validates a map of arguments against the rules
// For cases where an unknown command may be allowed because there may be other handlers for that command
//
//	failUnknownCommand = false
func ValidateArgs(args map[string]string, rule DWDirectiveRuleSpec, uniqueMap map[string]bool, failUnknownCommand bool) error {
	command := args["command"]

	// Determine the rules map for command
	rulesMap, err := BuildRulesMap(rule, command)
	if err != nil {
		// If the command is unsupported and we are supposed to fail in that case return error.
		// Otherwise just return nil to effectively skip the #DW
		// for info on errors.As() below see:
		// https://stackoverflow.com/questions/62441960/error-wrap-unwrap-type-checking-with-errors-is#62442136
		var unsupportedCommand *dwUnsupportedCommandErr
		if failUnknownCommand && errors.As(err, &unsupportedCommand) {
			return err
		}
		return nil
	}

	// Compile this regex outside the loop for better performance.
	var boolMatcher = regexp.MustCompile(`(?i)^(true|false)$`) // (?i) -> case-insensitve comparison

	// Create a map that maps a directive rule definition to an argument that correctly matches it
	// key: DWDirectiveRule	value: argument that matches that rule
	// Required to check that all DWDirectiveRuleDef's have been met
	argToRuleMap := map[DWDirectiveRuleDef]string{}

	// Iterate over all arguments and validate each based on the associated rule
	for k, v := range args {
		if k != "command" {
			rule, found := rulesMap[k]
			if !found {
				return errors.New("unsupported argument - " + k)
			}
			if rule.IsValueRequired && len(v) == 0 {
				return errors.New("malformed keyword[=value]: " + k + "=" + v)
			}
			switch rule.Type {
			case "integer":
				// i,err := strconv.ParseInt(v, 10, 64)
				i, err := strconv.Atoi(v)
				if err != nil {
					return errors.New("invalid integer argument: " + k + "=" + v)
				}
				if rule.Max != 0 && i > rule.Max {
					return errors.New("specified integer exceeds maximum " + strconv.Itoa(rule.Max) + ": " + k + "=" + v)
				}
				if rule.Min != 0 && i < rule.Min {
					return errors.New("specified integer smaller than minimum " + strconv.Itoa(rule.Min) + ": " + k + "=" + v)
				}
			case "bool":
				if rule.Pattern != "" {
					isok := boolMatcher.MatchString(v)
					if !isok {
						return errors.New("invalid bool argument: " + k + "=" + v)
					}
				}
			case "string":
				if rule.Pattern != "" {
					isok, err := regexp.MatchString(rule.Pattern, v)
					if !isok {
						if err != nil {
							return errors.New("invalid regexp in rule: " + rule.Pattern)
						}
						return errors.New("invalid argument: " + k + "=" + v)
					}
				}
			default:
				return errors.New("unsupported value type: " + rule.Type)
			}

			if rule.UniqueWithin != "" {
				_, ok := uniqueMap[rule.UniqueWithin+"/"+v]
				if ok {
					return fmt.Errorf("Value '%s' must be unique within '%s'", v, rule.UniqueWithin)
				}

				uniqueMap[rule.UniqueWithin+"/"+v] = true
			}

			// NOTE: We know that we don't have repeated arguments here because the arguments
			//       come to us in a map indexed by the argment name.
			argToRuleMap[rule] = k
		}
	}

	// Iterate over the rules to ensure all required rules have an argument
	for k, v := range rulesMap {
		// Ensure that each required rule has an argument
		if v.IsRequired {
			_, ok := argToRuleMap[v]
			if !ok {
				return errors.New("missing argument: " + k)
			}
		}
	}

	return nil
}

// ValidateDWDirective validates a set of #DW directives against a specified rule set
func ValidateDWDirective(rule DWDirectiveRuleSpec, dwd string, uniqueMap map[string]bool, failUnknownCommand bool) (bool, error) {

	// Build a map of the #DW commands and arguments
	argsMap, err := BuildArgsMap(dwd)
	if err != nil {
		return false, err
	}

	// If the command doesn't match...
	if argsMap["command"] != rule.Command {
		// If we need to fail unknown commands, return invalid command
		if failUnknownCommand {
			return false, nil
		}

		// Otherwise, we may have a new command that our code doesn't yet know
		// Don't bother checking the rest
		return true, nil
	}

	err = ValidateArgs(argsMap, rule, uniqueMap, failUnknownCommand)
	if err != nil {
		return false, err
	}

	return true, nil
}
